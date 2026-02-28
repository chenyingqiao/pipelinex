package local

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/chenyingqiao/pipelinex/executor"
)

// LocalExecutor 本地执行器实现
type LocalExecutor struct {
	workdir    string            // 工作目录
	env        map[string]string // 环境变量
	shell      string            // 使用的shell
	timeout    time.Duration     // 默认超时时间
	usePTY     bool              // 是否使用伪终端（支持交互式命令）
	ptyWidth   int               // 终端宽度
	ptyHeight  int               // 终端高度
	mu         sync.RWMutex
	currentCmd *exec.Cmd         // 当前执行的命令（用于取消）
}

// NewLocalExecutor 创建新的本地执行器
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{
		env:       make(map[string]string),
		shell:     detectDefaultShell(),
		timeout:   0, // 默认无超时
		usePTY:    false,
		ptyWidth:  80,
		ptyHeight: 24,
	}
}

// Prepare 准备本地执行环境
// 本地执行器不需要特殊的准备，只需要验证工作目录
func (l *LocalExecutor) Prepare(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 如果指定了工作目录，验证它存在
	if l.workdir != "" {
		info, err := os.Stat(l.workdir)
		if err != nil {
			return fmt.Errorf("workdir does not exist: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("workdir is not a directory: %s", l.workdir)
		}
	}

	// 验证shell可用
	if l.shell != "" {
		_, err := exec.LookPath(l.shell)
		if err != nil {
			// 尝试使用系统默认shell
			l.shell = detectDefaultShell()
		}
	}

	return nil
}

// Destruction 销毁本地执行环境
// 本地执行器不需要特殊的清理，但会终止正在运行的命令
func (l *LocalExecutor) Destruction(ctx context.Context) error {
	l.killCurrentProcess()
	return nil
}

// Transfer 接收命令并执行
// 只支持 string 类型的命令
//
// 当 ctx 被取消时，会立即停止执行新命令，并终止当前正在执行的进程
func (l *LocalExecutor) Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any) {
	// 创建一个可取消的内部上下文，用于控制当前命令的执行
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 启动一个 goroutine 监听外部上下文取消
	// 当外部上下文被取消时，取消内部上下文并终止当前进程
	go func() {
		<-ctx.Done()
		// 外部上下文被取消，取消内部上下文并终止当前进程
		cancel()
		l.killCurrentProcess()
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
			l.executeCommandStreaming(execCtx, v, resultChan)
		default:
			resultChan <- fmt.Errorf("unsupported data type: %T", data)
		}
	}
}

// killCurrentProcess 终止当前正在执行的进程
func (l *LocalExecutor) killCurrentProcess() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentCmd != nil && l.currentCmd.Process != nil {
		// 先尝试发送中断信号（Unix）或 Ctrl+Break（Windows）
		if err := l.currentCmd.Process.Signal(os.Interrupt); err != nil {
			// 如果优雅终止失败，强制终止
			_ = l.currentCmd.Process.Kill()
		}
	}
}

// executeCommandStreaming 执行命令并实时流式输出
func (l *LocalExecutor) executeCommandStreaming(ctx context.Context, command string, resultChan chan<- any) {
	startTime := time.Now()

	// 创建带超时的上下文
	execCtx := ctx
	if l.timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, l.timeout)
		defer cancel()
	}

	err := l.executeCommandWithStreaming(execCtx, command, func(data []byte) {
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

// executeCommandWithStreaming 执行命令并实时输出
func (l *LocalExecutor) executeCommandWithStreaming(ctx context.Context, command string, outputCallback func([]byte)) error {
	l.mu.Lock()

	// 创建命令
	cmd := l.createCommand(ctx, command)
	l.currentCmd = cmd

	// 设置工作目录
	if l.workdir != "" {
		cmd.Dir = l.workdir
	}

	// 设置环境变量
	cmd.Env = l.buildEnvList()

	l.mu.Unlock()

	// 获取stdout和stderr管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// 使用 WaitGroup 等待两个输出流读取完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 读取stdout
	go func() {
		defer wg.Done()
		l.streamOutput(stdout, outputCallback)
	}()

	// 读取stderr
	go func() {
		defer wg.Done()
		l.streamOutput(stderr, outputCallback)
	}()

	// 等待输出读取完成
	wg.Wait()

	// 等待命令完成
	err = cmd.Wait()

	l.mu.Lock()
	l.currentCmd = nil
	l.mu.Unlock()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out: %w", err)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("command exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// streamOutput 读取输出并回调
func (l *LocalExecutor) streamOutput(reader io.Reader, callback func([]byte)) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 && callback != nil {
			callback(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

// streamOutputLine 按行读取输出并回调
func (l *LocalExecutor) streamOutputLine(reader io.Reader, callback func([]byte)) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if callback != nil {
			callback(append(scanner.Bytes(), '\n'))
		}
	}
}

// createCommand 根据操作系统创建命令
func (l *LocalExecutor) createCommand(ctx context.Context, command string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		// Windows使用cmd.exe
		if l.shell == "powershell" || l.shell == "pwsh" {
			return exec.CommandContext(ctx, l.shell, "-Command", command)
		}
		return exec.CommandContext(ctx, "cmd", "/C", command)
	default:
		// Unix-like系统使用sh或bash
		shell := l.shell
		if shell == "" {
			shell = "/bin/sh"
		}
		return exec.CommandContext(ctx, shell, "-c", command)
	}
}

// buildEnvList 构建环境变量列表
func (l *LocalExecutor) buildEnvList() []string {
	// 从当前进程环境变量开始
	envList := os.Environ()

	// 添加自定义环境变量（覆盖现有变量）
	for k, v := range l.env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	return envList
}

// setWorkdir 设置工作目录
func (l *LocalExecutor) setWorkdir(workdir string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.workdir = workdir
}

// setEnv 设置环境变量
func (l *LocalExecutor) setEnv(key, value string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.env[key] = value
}

// setShell 设置shell
func (l *LocalExecutor) setShell(shell string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shell = shell
}

// setTimeout 设置默认超时
func (l *LocalExecutor) setTimeout(timeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeout = timeout
}

// setPTY 设置是否使用伪终端
func (l *LocalExecutor) setPTY(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.usePTY = enabled
}

// setPTYSize 设置终端尺寸
func (l *LocalExecutor) setPTYSize(width, height int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ptyWidth = width
	l.ptyHeight = height
}

// GetWorkdir 获取工作目录
func (l *LocalExecutor) GetWorkdir() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.workdir
}

// GetShell 获取当前shell
func (l *LocalExecutor) GetShell() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.shell
}

// detectDefaultShell 检测系统默认shell
func detectDefaultShell() string {
	switch runtime.GOOS {
	case "windows":
		// Windows优先使用PowerShell，回退到cmd
		if _, err := exec.LookPath("pwsh"); err == nil {
			return "pwsh"
		}
		if _, err := exec.LookPath("powershell"); err == nil {
			return "powershell"
		}
		return "cmd"
	default:
		// Unix-like系统优先使用bash，回退到sh
		if _, err := exec.LookPath("bash"); err == nil {
			return "/bin/bash"
		}
		return "/bin/sh"
	}
}

// isShellAvailable 检查shell是否可用
func isShellAvailable(shell string) bool {
	_, err := exec.LookPath(shell)
	return err == nil
}

// 确保LocalExecutor实现了Executor接口
var _ executor.Executor = (*LocalExecutor)(nil)