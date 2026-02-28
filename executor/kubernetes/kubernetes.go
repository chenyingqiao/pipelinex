package kubernetes

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/chenyingqiao/pipelinex/executor"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// KubernetesExecutor Kubernetes执行器实现
type KubernetesExecutor struct {
	client          kubernetes.Interface
	restConfig      *rest.Config
	namespace       string
	podName         string
	image           string
	workdir         string
	env             map[string]string
	volumes         []corev1.Volume
	volumeMounts    []corev1.VolumeMount
	serviceAccount  string
	tty             bool
	ttyHeight       uint
	ttyWidth        uint
	podReadyTimeout time.Duration // Pod 就绪等待超时时间
	// 用于取消当前执行的命令
	currentExecCancel context.CancelFunc
	mu                sync.RWMutex
}

// NewKubernetesExecutor 创建新的Kubernetes执行器
// 使用当前环境的kubeconfig
func NewKubernetesExecutor() (*KubernetesExecutor, error) {
	// 尝试从当前环境加载kubeconfig
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesExecutor{
		client:          clientset,
		restConfig:      config,
		env:             make(map[string]string),
		volumes:         make([]corev1.Volume, 0),
		volumeMounts:    make([]corev1.VolumeMount, 0),
		namespace:       "default",
		podReadyTimeout: 60 * time.Second, // 默认 60 秒
	}, nil
}

// NewKubernetesExecutorWithConfig 使用指定的rest.Config创建执行器
func NewKubernetesExecutorWithConfig(config *rest.Config) (*KubernetesExecutor, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesExecutor{
		client:          clientset,
		restConfig:      config,
		env:             make(map[string]string),
		volumes:         make([]corev1.Volume, 0),
		volumeMounts:    make([]corev1.VolumeMount, 0),
		namespace:       "default",
		podReadyTimeout: 60 * time.Second, // 默认 60 秒
	}, nil
}

// NewKubernetesExecutorWithClient 使用指定的客户端创建执行器
func NewKubernetesExecutorWithClient(client kubernetes.Interface, config *rest.Config, namespace string) *KubernetesExecutor {
	if namespace == "" {
		namespace = "default"
	}

	return &KubernetesExecutor{
		client:       client,
		restConfig:   config,
		namespace:    namespace,
		env:          make(map[string]string),
		volumes:      make([]corev1.Volume, 0),
		volumeMounts: make([]corev1.VolumeMount, 0),
	}
}

// Prepare 准备Kubernetes环境
// 创建并等待Pod运行
func (k *KubernetesExecutor) Prepare(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// 如果没有指定镜像，使用默认镜像
	if k.image == "" {
		k.image = "alpine:latest"
	}

	// 生成唯一的pod名称
	k.podName = fmt.Sprintf("pipelinex-%d", time.Now().UnixNano())

	// 构建Pod配置
	pod := k.buildPodSpec()

	// 创建Pod
	createdPod, err := k.client.CoreV1().Pods(k.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	k.podName = createdPod.Name

	// 等待Pod启动完成
	if err := k.waitForPodRunning(ctx); err != nil {
		return fmt.Errorf("pod failed to start: %w", err)
	}

	return nil
}

// Destruction 销毁Kubernetes环境
// 删除Pod
func (k *KubernetesExecutor) Destruction(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.podName == "" {
		return nil
	}

	// 删除Pod
	deletePolicy := metav1.DeletePropagationBackground
	gracePeriod := int64(10)
	err := k.client.CoreV1().Pods(k.namespace).Delete(ctx, k.podName, metav1.DeleteOptions{
		PropagationPolicy:  &deletePolicy,
		GracePeriodSeconds: &gracePeriod,
	})

	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	k.podName = ""
	return nil
}

// Transfer 在Kubernetes Pod中执行命令
// 只支持 string 类型的命令
//
// 当 ctx 被取消时，会立即停止执行新命令，并终止当前正在执行的命令
func (k *KubernetesExecutor) Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any) {
	// 创建一个可取消的内部上下文，用于控制当前命令的执行
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 启动一个 goroutine 监听外部上下文取消
	go func() {
		<-ctx.Done()
		cancel()
	}()

	for data := range commandChan {
		// 检查上下文是否已取消
		select {
		case <-execCtx.Done():
			return
		default:
		}

		// 处理不同类型的数据
		switch v := data.(type) {
		case string:
			// 执行命令（实时输出）
			k.executeCommandStreaming(execCtx, v, resultChan)
		default:
			resultChan <- fmt.Errorf("unsupported data type: %T", data)
		}
	}
}

// buildPodSpec 构建Pod配置
func (k *KubernetesExecutor) buildPodSpec() *corev1.Pod {
	// 构建环境变量
	envVars := make([]corev1.EnvVar, 0, len(k.env))
	for key, value := range k.env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// 构建容器配置
	container := corev1.Container{
		Name:    "executor",
		Image:   k.image,
		Command: []string{"sleep", "3600"},
		Env:     envVars,
	}

	// 设置工作目录
	if k.workdir != "" {
		container.WorkingDir = k.workdir
	}

	// 设置卷挂载
	if len(k.volumeMounts) > 0 {
		container.VolumeMounts = k.volumeMounts
	}

	// 构建Pod配置
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   k.podName,
			Labels: map[string]string{
				"app":       "pipelinex",
				"component": "executor",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{container},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// 设置ServiceAccount
	if k.serviceAccount != "" {
		pod.Spec.ServiceAccountName = k.serviceAccount
	}

	// 设置卷
	if len(k.volumes) > 0 {
		pod.Spec.Volumes = k.volumes
	}

	return pod
}

// waitForPodRunning 等待Pod进入Running状态
func (k *KubernetesExecutor) waitForPodRunning(ctx context.Context) error {
	k.mu.RLock()
	timeout := k.podReadyTimeout
	k.mu.RUnlock()

	// 如果未设置，使用默认 60 秒
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	// 创建带超时的上下文
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		pod, err := k.client.CoreV1().Pods(k.namespace).Get(waitCtx, k.podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			// 检查容器是否就绪
			if len(pod.Status.ContainerStatuses) > 0 {
				containerReady := pod.Status.ContainerStatuses[0].Ready
				if containerReady || pod.Status.ContainerStatuses[0].State.Running != nil {
					return nil
				}
			}
		case corev1.PodFailed, corev1.PodSucceeded:
			return fmt.Errorf("pod exited with status: %s", pod.Status.Phase)
		}

		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout waiting for pod to start after %v", timeout)
			}
			return ctx.Err()
		case <-ticker.C:
			// 继续下一轮检查
		}
	}
}

// executeCommandStreaming 执行命令并实时流式输出
func (k *KubernetesExecutor) executeCommandStreaming(ctx context.Context, command string, resultChan chan<- any) {
	startTime := time.Now()

	err := k.executeCommandInPodStreaming(ctx, command, func(data []byte) {
		resultChan <- data
	})

	// 发送最终结果
	resultChan <- &executor.StepResult{
		Command:    command,
		Output:     "",
		Error:      err,
		StartTime:  startTime,
		FinishTime: time.Now(),
	}
}

// executeCommandInPodStreaming 在Pod中执行命令并实时流式输出
// 当 ctx 被取消时，会向进程发送 Ctrl+C 信号 (\x03)
func (k *KubernetesExecutor) executeCommandInPodStreaming(ctx context.Context, command string, outputCallback func([]byte)) error {
	k.mu.RLock()
	podName := k.podName
	namespace := k.namespace
	useTTY := k.tty
	ttyWidth := k.ttyWidth
	ttyHeight := k.ttyHeight
	k.mu.RUnlock()

	if podName == "" {
		return fmt.Errorf("pod not prepared")
	}

	// 检测shell类型
	shell := k.detectShell()

	// 构建exec请求
	req := k.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", "executor")

	// 添加执行参数
	req = req.Param("command", shell)
	req = req.Param("command", "-c")
	req = req.Param("command", command)

	// 设置TTY和流选项
	req = req.Param("stdout", "true")
	req = req.Param("stderr", "true")
	// 必须启用stdin才能发送Ctrl+C
	req = req.Param("stdin", "true")
	req = req.Param("tty", fmt.Sprintf("%v", useTTY))

	// 创建执行器
	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// 创建流式输出器
	streamer := &execStreamer{
		callback: outputCallback,
		useTTY:   useTTY,
	}

	// 创建stdin pipe，用于发送Ctrl+C
	stdinReader, stdinWriter := io.Pipe()

	// 创建一个内部可取消的上下文
	execCtx, execCancel := context.WithCancel(ctx)
	defer execCancel()

	// 注册取消函数，供外部调用
	k.mu.Lock()
	k.currentExecCancel = func() {
		// 发送 Ctrl+C (\x03)
		stdinWriter.Write([]byte{0x03})
		stdinWriter.Close()
		execCancel()
	}
	k.mu.Unlock()

	// 监听上下文取消，发送Ctrl+C
	go func() {
		<-ctx.Done()
		k.mu.RLock()
		cancel := k.currentExecCancel
		k.mu.RUnlock()
		if cancel != nil {
			cancel()
		}
	}()

	// 执行命令
	streamOptions := remotecommand.StreamOptions{
		Stdin:  stdinReader,
		Stdout: streamer,
		Stderr: streamer,
		Tty:    useTTY,
	}

	// 如果启用TTY，设置终端大小
	if useTTY && ttyWidth > 0 && ttyHeight > 0 {
		streamOptions.TerminalSizeQueue = &fixedTerminalSize{
			width:  uint16(ttyWidth),
			height: uint16(ttyHeight),
		}
	}

	err = exec.StreamWithContext(execCtx, streamOptions)

	// 清理
	stdinWriter.Close()
	k.mu.Lock()
	k.currentExecCancel = nil
	k.mu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if execCtx.Err() != nil {
			return execCtx.Err()
		}
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// execStreamer 执行流输出器
type execStreamer struct {
	callback func([]byte)
	useTTY   bool
	data     []byte
}

// Write 实现io.Writer接口
func (s *execStreamer) Write(p []byte) (n int, err error) {
	if s.callback != nil {
		s.callback(p)
	}
	return len(p), nil
}

// Read 实现io.Reader接口（用于stdin，这里不需要）
func (s *execStreamer) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

// fixedTerminalSize 固定终端大小
type fixedTerminalSize struct {
	width  uint16
	height uint16
	done   bool
}

// Next 实现TerminalSizeQueue接口
func (f *fixedTerminalSize) Next() *remotecommand.TerminalSize {
	if f.done {
		return nil
	}
	f.done = true
	return &remotecommand.TerminalSize{
		Width:  f.width,
		Height: f.height,
	}
}

// detectShell 检测容器中的shell
func (k *KubernetesExecutor) detectShell() string {
	// 根据镜像类型选择shell
	image := k.image
	if containsIgnoreCase(image, "alpine") || containsIgnoreCase(image, "busybox") {
		return "/bin/sh"
	}
	return "/bin/bash"
}

// containsIgnoreCase 不区分大小写包含检查
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstrIgnoreCase(s, substr)
}

func containsSubstrIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// setImage 设置镜像
func (k *KubernetesExecutor) setImage(image string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.image = image
}

// setWorkdir 设置工作目录
func (k *KubernetesExecutor) setWorkdir(workdir string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.workdir = workdir
}

// setEnv 设置环境变量
func (k *KubernetesExecutor) setEnv(key, value string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.env[key] = value
}

// setNamespace 设置命名空间
func (k *KubernetesExecutor) setNamespace(namespace string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.namespace = namespace
}

// setServiceAccount 设置ServiceAccount
func (k *KubernetesExecutor) setServiceAccount(sa string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.serviceAccount = sa
}

// setVolume 添加卷挂载（ConfigMap或Secret）
func (k *KubernetesExecutor) setVolume(volume corev1.Volume, mount corev1.VolumeMount) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.volumes = append(k.volumes, volume)
	k.volumeMounts = append(k.volumeMounts, mount)
}

// setTTY 设置是否启用TTY模式
func (k *KubernetesExecutor) setTTY(enabled bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tty = enabled
}

// setTTYSize 设置TTY终端尺寸
func (k *KubernetesExecutor) setTTYSize(width, height uint) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.ttyWidth = width
	k.ttyHeight = height
}

// setPodReadyTimeout 设置Pod就绪等待超时时间
func (k *KubernetesExecutor) setPodReadyTimeout(timeout time.Duration) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.podReadyTimeout = timeout
}

// GetPodName 获取Pod名称
func (k *KubernetesExecutor) GetPodName() string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.podName
}

// GetNamespace 获取命名空间
func (k *KubernetesExecutor) GetNamespace() string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.namespace
}

// 确保KubernetesExecutor实现了Executor接口
var _ executor.Executor = (*KubernetesExecutor)(nil)
