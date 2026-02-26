package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/chenyingqiao/pipelinex"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// DockerExecutor Docker执行器实现
type DockerExecutor struct {
	client      *client.Client
	containerID string
	image       string
	workdir     string
	env         map[string]string
	volumes     map[string]string
	network     string
	registry    string
	tty         bool // 是否启用 TTY 模式
	ttyHeight   uint // TTY 终端高度
	ttyWidth    uint // TTY 终端宽度
	mu          sync.RWMutex
}

// callbackWriter 自定义 Writer 用于实时回调输出
type callbackWriter struct {
	callback func([]byte)
}

// NewDockerExecutor 创建新的Docker执行器
func NewDockerExecutor() (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerExecutor{
		client:  cli,
		env:     make(map[string]string),
		volumes: make(map[string]string),
	}, nil
}

// NewDockerExecutorWithClient 使用指定的Docker客户端创建执行器
func NewDockerExecutorWithClient(cli *client.Client) *DockerExecutor {
	return &DockerExecutor{
		client:  cli,
		env:     make(map[string]string),
		volumes: make(map[string]string),
	}
}

// Prepare 准备Docker环境
// 1. 检查/拉取镜像
// 2. 创建并启动容器
func (d *DockerExecutor) Prepare(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 如果没有指定镜像，使用默认镜像
	if d.image == "" {
		d.image = "alpine:latest"
	}

	// 解析镜像名称（处理registry）
	fullImage := d.resolveImageName()

	// 检查镜像是否存在，不存在则拉取
	if err := d.pullImageIfNeeded(ctx, fullImage); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// 构建容器配置
	containerConfig := &container.Config{
		Image:        fullImage,
		Cmd:          []string{"sleep", "3600"},
		WorkingDir:   d.workdir,
		Env:          d.buildEnvList(),
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	// 构建主机配置
	hostConfig := &container.HostConfig{
		Mounts:     d.buildMounts(),
		AutoRemove: false,
	}

	// 设置网络模式
	if d.network != "" {
		hostConfig.NetworkMode = container.NetworkMode(d.network)
	}

	// 创建容器
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, fmt.Sprintf("pipelinex-%d", time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	d.containerID = resp.ID

	// 启动容器
	if err := d.client.ContainerStart(ctx, d.containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// 等待容器启动完成
	if err := d.waitForContainer(ctx); err != nil {
		return fmt.Errorf("container failed to start: %w", err)
	}

	return nil
}

// Destruction 销毁Docker环境
// 停止并删除容器
func (d *DockerExecutor) Destruction(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.containerID == "" {
		return nil
	}

	// 停止容器
	timeout := 10
	_ = d.client.ContainerStop(ctx, d.containerID, container.StopOptions{
		Timeout: &timeout,
	})

	// 删除容器
	if err := d.client.ContainerRemove(ctx, d.containerID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	d.containerID = ""
	return nil
}

// Transfer 在Docker容器中执行命令
// in 接收执行数据（包括步骤信息），out 发送执行结果
//
// 当 ctx 被取消时，会立即停止执行新命令，并终止当前正在容器内执行的命令
func (d *DockerExecutor) Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any) {
	// 创建一个可取消的内部上下文，用于控制当前命令的执行
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 启动一个 goroutine 监听外部上下文取消
	// 当外部上下文被取消时，取消内部上下文
	// 上下文取消会触发 executeCommandInContainerStreaming 中的连接关闭
	// 从而使容器内进程收到 SIGHUP 信号而终止
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
		case pipelinex.Step:
			// 执行步骤（实时输出）
			d.executeStepStreaming(execCtx, v, resultChan)
		case []pipelinex.Step:
			// 执行多个步骤（实时输出）
			for _, step := range v {
				// 检查上下文是否已取消
				select {
				case <-execCtx.Done():
					return
				default:
				}
				d.executeStepStreaming(execCtx, step, resultChan)
			}
		case string:
			// 执行命令（实时输出）
			d.executeCommandStreaming(execCtx, v, resultChan)
		default:
			resultChan <- fmt.Errorf("unsupported data type: %T", data)
		}
	}
}

// executeStep 执行单个步骤
func (d *DockerExecutor) executeStep(ctx context.Context, step pipelinex.Step) *StepResult {
	startTime := time.Now()

	output, err := d.executeCommandInContainer(ctx, step.Run)

	return &StepResult{
		StepName:   step.Name,
		Command:    step.Run,
		Output:     output,
		Error:      err,
		StartTime:  startTime,
		FinishTime: time.Now(),
	}
}

// executeCommand 执行命令
func (d *DockerExecutor) executeCommand(ctx context.Context, command string) *StepResult {
	startTime := time.Now()

	output, err := d.executeCommandInContainer(ctx, command)

	return &StepResult{
		Command:    command,
		Output:     output,
		Error:      err,
		StartTime:  startTime,
		FinishTime: time.Now(),
	}
}

// executeCommandInContainer 在容器中执行命令
func (d *DockerExecutor) executeCommandInContainer(ctx context.Context, command string) (string, error) {
	d.mu.RLock()
	containerID := d.containerID
	d.mu.RUnlock()

	if containerID == "" {
		return "", fmt.Errorf("container not prepared")
	}

	// 检测shell类型
	shell := d.detectShell()

	// 创建执行配置
	execConfig := container.ExecOptions{
		Cmd:          []string{shell, "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	// 创建执行
	execResp, err := d.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	// 附加到执行
	attachResp, err := d.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	// 读取输出
	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read output: %w", err)
	}

	// 等待执行完成
	for {
		inspectResp, err := d.client.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return "", fmt.Errorf("failed to inspect exec: %w", err)
		}

		if !inspectResp.Running {
			if inspectResp.ExitCode != 0 {
				return stdout.String() + stderr.String(), fmt.Errorf("command exited with code %d: %s", inspectResp.ExitCode, stderr.String())
			}
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return stdout.String(), nil
}

// executeStepStreaming 执行步骤并实时流式输出
func (d *DockerExecutor) executeStepStreaming(ctx context.Context, step pipelinex.Step, resultChan chan<- any) {
	startTime := time.Now()
	var execErr error

	// 实时执行命令
	err := d.executeCommandInContainerStreaming(ctx, step.Run, func(data []byte) {
		resultChan <- data
	})

	if err != nil {
		execErr = err
	}

	// 发送最终结果
	resultChan <- &StepResult{
		StepName:   step.Name,
		Command:    step.Run,
		Output:     "",
		Error:      execErr,
		StartTime:  startTime,
		FinishTime: time.Now(),
	}
}

// executeCommandStreaming 执行命令并实时流式输出
func (d *DockerExecutor) executeCommandStreaming(ctx context.Context, command string, resultChan chan<- any) {
	startTime := time.Now()

	err := d.executeCommandInContainerStreaming(ctx, command, func(data []byte) {
		resultChan <- data
	})

	// 发送最终结果
	resultChan <- &StepResult{
		Command:    command,
		Output:     "",
		Error:      err,
		StartTime:  startTime,
		FinishTime: time.Now(),
	}
}

// executeCommandInContainerStreaming 在容器中执行命令并实时流式输出
func (d *DockerExecutor) executeCommandInContainerStreaming(ctx context.Context, command string, outputCallback func([]byte)) error {
	containerID, useTTY, err := d.getContainerInfo()
	if err != nil {
		return err
	}

	execID, attachResp, err := d.createAndAttachExec(ctx, containerID, command, useTTY)
	if err != nil {
		return err
	}
	defer attachResp.Close()

	outputDone := d.startOutputStream(ctx, attachResp, outputCallback, useTTY)

	return d.waitForExecCompletion(ctx, execID, attachResp, outputDone)
}

// getContainerInfo 获取容器信息
func (d *DockerExecutor) getContainerInfo() (string, bool, error) {
	d.mu.RLock()
	containerID := d.containerID
	useTTY := d.tty
	d.mu.RUnlock()

	if containerID == "" {
		return "", false, fmt.Errorf("container not prepared")
	}
	return containerID, useTTY, nil
}

// createAndAttachExec 创建并附加到 exec
func (d *DockerExecutor) createAndAttachExec(ctx context.Context, containerID, command string, useTTY bool) (string, types.HijackedResponse, error) {
	shell := d.detectShell()

	execConfig := container.ExecOptions{
		Cmd:          []string{shell, "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          useTTY,
	}

	execResp, err := d.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", types.HijackedResponse{}, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := d.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: useTTY})
	if err != nil {
		return "", types.HijackedResponse{}, fmt.Errorf("failed to attach to exec: %w", err)
	}

	return execResp.ID, attachResp, nil
}

// startOutputStream 启动输出流读取
func (d *DockerExecutor) startOutputStream(ctx context.Context, attachResp types.HijackedResponse, outputCallback func([]byte), useTTY bool) chan error {
	outputDone := make(chan error, 1)

	go func() {
		if useTTY {
			outputDone <- d.readTTYOutput(ctx, attachResp.Reader, outputCallback)
		} else {
			outputDone <- d.readNonTTYOutput(ctx, attachResp, outputCallback)
		}
	}()

	return outputDone
}

// readTTYOutput 读取 TTY 模式输出
func (d *DockerExecutor) readTTYOutput(ctx context.Context, reader io.Reader, outputCallback func([]byte)) error {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 && outputCallback != nil {
			outputCallback(buf[:n])
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
	}
}

// readNonTTYOutput 读取非 TTY 模式输出
func (d *DockerExecutor) readNonTTYOutput(ctx context.Context, attachResp types.HijackedResponse, outputCallback func([]byte)) error {
	writer := &callbackWriter{callback: outputCallback}
	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if conn, ok := attachResp.Conn.(interface{ SetReadDeadline(time.Time) error }); ok {
			_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		}

		n, err := attachResp.Reader.Read(buf)
		if n > 0 {
			_, _ = writer.Write(buf[:n])
		}
		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			if err != io.EOF {
				return err
			}
			return nil
		}
	}
}

// waitForExecCompletion 等待执行完成
func (d *DockerExecutor) waitForExecCompletion(ctx context.Context, execID string, attachResp types.HijackedResponse, outputDone chan error) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			attachResp.Close()
			<-outputDone
			return ctx.Err()
		case err := <-outputDone:
			if err == context.Canceled || err == context.DeadlineExceeded {
				return ctx.Err()
			}
		case <-ticker.C:
		}

		inspectResp, err := d.client.ContainerExecInspect(ctx, execID)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("failed to inspect exec: %w", err)
		}

		if !inspectResp.Running {
			if inspectResp.ExitCode != 0 {
				<-outputDone
				return fmt.Errorf("command exited with code %d", inspectResp.ExitCode)
			}
			break
		}
	}

	return <-outputDone
}

// 为 callbackWriter 实现 Write 方法
func (w *callbackWriter) Write(p []byte) (n int, err error) {
	if w.callback != nil {
		w.callback(p)
	}
	return len(p), nil
}

// detectShell 检测容器中的shell
func (d *DockerExecutor) detectShell() string {
	// 根据镜像类型选择shell
	image := strings.ToLower(d.image)
	if strings.Contains(image, "alpine") || strings.Contains(image, "busybox") {
		return "/bin/sh"
	}
	return "/bin/bash"
}

// pullImageIfNeeded 检查并拉取镜像
func (d *DockerExecutor) pullImageIfNeeded(ctx context.Context, imageName string) error {
	// 检查镜像是否存在
	_, _, err := d.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// 镜像已存在
		return nil
	}

	// 镜像不存在，需要拉取
	reader, err := d.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// 等待拉取完成（读取所有输出）
	_, _ = io.Copy(io.Discard, reader)

	return nil
}

// waitForContainer 等待容器启动完成
func (d *DockerExecutor) waitForContainer(ctx context.Context) error {
	for i := 0; i < 30; i++ {
		containerJSON, err := d.client.ContainerInspect(ctx, d.containerID)
		if err != nil {
			return err
		}

		if containerJSON.State.Running {
			return nil
		}

		if containerJSON.State.ExitCode != 0 {
			return fmt.Errorf("container exited with code %d", containerJSON.State.ExitCode)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for container to start")
}

// resolveImageName 解析完整的镜像名称
func (d *DockerExecutor) resolveImageName() string {
	if d.registry == "" || strings.Contains(d.image, "/") {
		return d.image
	}
	return fmt.Sprintf("%s/%s", d.registry, d.image)
}

// buildEnvList 构建环境变量列表
func (d *DockerExecutor) buildEnvList() []string {
	envList := make([]string, 0, len(d.env))
	for k, v := range d.env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	return envList
}

// buildMounts 构建挂载配置
func (d *DockerExecutor) buildMounts() []mount.Mount {
	mounts := make([]mount.Mount, 0, len(d.volumes))
	for hostPath, containerPath := range d.volumes {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: hostPath,
			Target: containerPath,
		})
	}
	return mounts
}

// copyToContainer 复制文件到容器
func (d *DockerExecutor) copyToContainer(ctx context.Context, localPath, containerPath string) error {
	d.mu.RLock()
	containerID := d.containerID
	d.mu.RUnlock()

	if containerID == "" {
		return fmt.Errorf("container not prepared")
	}

	// TODO: 实现文件复制
	return fmt.Errorf("not implemented")
}

// copyFromContainer 从容器复制文件
func (d *DockerExecutor) copyFromContainer(ctx context.Context, containerPath, localPath string) error {
	d.mu.RLock()
	containerID := d.containerID
	d.mu.RUnlock()

	if containerID == "" {
		return fmt.Errorf("container not prepared")
	}

	// TODO: 实现文件复制
	return fmt.Errorf("not implemented")
}

// setImage 设置镜像
func (d *DockerExecutor) setImage(image string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.image = image
}

// setWorkdir 设置工作目录
func (d *DockerExecutor) setWorkdir(workdir string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.workdir = workdir
}

// setEnv 设置环境变量
func (d *DockerExecutor) setEnv(key, value string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.env[key] = value
}

// setVolume 设置卷挂载
func (d *DockerExecutor) setVolume(hostPath, containerPath string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.volumes[hostPath] = containerPath
}

// setNetwork 设置网络
func (d *DockerExecutor) setNetwork(network string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.network = network
}

// setRegistry 设置镜像仓库
func (d *DockerExecutor) setRegistry(registry string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.registry = registry
}

// setTTY 设置是否启用 TTY 模式
func (d *DockerExecutor) setTTY(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tty = enabled
}

// setTTYSize 设置 TTY 终端尺寸
func (d *DockerExecutor) setTTYSize(width, height uint) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ttyWidth = width
	d.ttyHeight = height
}

// GetContainerID 获取容器ID
func (d *DockerExecutor) GetContainerID() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.containerID
}

// StepResult 步骤执行结果
type StepResult struct {
	StepName   string
	Command    string
	Output     string
	Error      error
	StartTime  time.Time
	FinishTime time.Time
}

// 确保DockerExecutor实现了Executor接口
var _ pipelinex.Executor = (*DockerExecutor)(nil)
