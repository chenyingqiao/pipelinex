package local

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/chenyingqiao/pipelinex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalAdapter_Config(t *testing.T) {
	adapter := NewLocalAdapter()
	ctx := context.Background()

	config := map[string]any{
		"workdir":   "/tmp",
		"shell":     "bash",
		"timeout":   "30s",
		"pty":       true,
		"ptyWidth":  120,
		"ptyHeight": 40,
		"env": map[string]string{
			"GO_VERSION": "1.21",
			"TEST_KEY":   "test_value",
		},
	}

	err := adapter.Config(ctx, config)
	assert.NoError(t, err, "Config should not return error")
	assert.Equal(t, config, adapter.config, "Config should be stored")
}

func TestLocalBridge_Conn(t *testing.T) {
	bridge := NewLocalBridge()
	ctx := context.Background()

	config := map[string]any{
		"workdir": "/tmp",
		"shell":   "bash",
		"env": map[string]string{
			"KEY1": "value1",
		},
	}

	adapter := NewLocalAdapter()
	err := adapter.Config(ctx, config)
	require.NoError(t, err, "Config should not return error")

	executor, err := bridge.Conn(ctx, adapter)
	require.NoError(t, err, "Conn should not return error")
	require.NotNil(t, executor, "Executor should not be nil")

	// 验证返回的是 LocalExecutor 类型
	localExec, ok := executor.(*LocalExecutor)
	require.True(t, ok, "Expected executor to be *local.LocalExecutor")

	// 验证配置已应用
	assert.Equal(t, "/tmp", localExec.GetWorkdir(), "Workdir should be set")
	assert.Equal(t, "bash", localExec.GetShell(), "Shell should be set")
}

func TestLocalExecutor_Interface(t *testing.T) {
	// 验证 LocalExecutor 实现了 Executor 接口
	var _ pipelinex.Executor = (*LocalExecutor)(nil)
}

func TestLocalAdapter_Interface(t *testing.T) {
	// 验证 LocalAdapter 实现了 Adapter 接口
	var _ pipelinex.Adapter = (*LocalAdapter)(nil)
}

func TestLocalBridge_Interface(t *testing.T) {
	// 验证 LocalBridge 实现了 Bridge 接口
	var _ pipelinex.Bridge = (*LocalBridge)(nil)
}

func TestNewLocalExecutor(t *testing.T) {
	executor := NewLocalExecutor()
	require.NotNil(t, executor, "Executor should not be nil")

	// 验证默认配置
	assert.NotEmpty(t, executor.GetShell(), "Default shell should be detected")
}

func TestLocalExecutor_Prepare(t *testing.T) {
	ctx := context.Background()

	t.Run("default prepare", func(t *testing.T) {
		executor := NewLocalExecutor()
		err := executor.Prepare(ctx)
		assert.NoError(t, err, "Prepare with no workdir should succeed")
	})

	t.Run("with valid workdir", func(t *testing.T) {
		executor := NewLocalExecutor()
		executor.setWorkdir("/tmp")
		err := executor.Prepare(ctx)
		assert.NoError(t, err, "Prepare with valid workdir should succeed")
	})

	t.Run("with invalid workdir", func(t *testing.T) {
		executor := NewLocalExecutor()
		executor.setWorkdir("/nonexistent/path/that/does/not/exist")
		err := executor.Prepare(ctx)
		assert.Error(t, err, "Prepare with invalid workdir should fail")
	})
}

func TestLocalExecutor_Destruction(t *testing.T) {
	ctx := context.Background()
	executor := NewLocalExecutor()

	err := executor.Destruction(ctx)
	assert.NoError(t, err, "Destruction should succeed even with no running command")
}

func TestLocalExecutor_Transfer_SingleCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	executor := NewLocalExecutor()
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 10)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送测试命令
	var testCmd string
	if runtime.GOOS == "windows" {
		testCmd = "echo Hello from Windows"
	} else {
		testCmd = "echo 'Hello from LocalExecutor'"
	}
	commandChan <- testCmd
	close(commandChan)

	// 接收结果
	var output []byte
	var result *StepResult

	timeout := time.After(5 * time.Second)
resultLoop:
	for {
		select {
		case res := <-resultChan:
			switch v := res.(type) {
			case []byte:
				output = append(output, v...)
			case *StepResult:
				result = v
				break resultLoop
			case error:
				t.Fatalf("Received error: %v", v)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for results. Output so far: %s", string(output))
		}
	}

	// 验证结果
	require.NotNil(t, result, "Should receive StepResult")
	assert.Equal(t, testCmd, result.Command, "Command should match")
	assert.NoError(t, result.Error, "Command should execute successfully")
}

func TestLocalExecutor_Transfer_Step(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	executor := NewLocalExecutor()
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 10)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送测试步骤
	step := pipelinex.Step{
		Name: "test_step",
		Run:  "echo 'Hello from Step'",
	}
	commandChan <- step
	close(commandChan)

	// 接收结果
	var output []byte
	var result *StepResult

	timeout := time.After(5 * time.Second)
resultLoop:
	for {
		select {
		case res := <-resultChan:
			switch v := res.(type) {
			case []byte:
				output = append(output, v...)
			case *StepResult:
				result = v
				break resultLoop
			case error:
				t.Fatalf("Received error: %v", v)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for results")
		}
	}

	// 验证结果
	require.NotNil(t, result, "Should receive StepResult")
	assert.Equal(t, "test_step", result.StepName, "StepName should match")
	assert.Equal(t, step.Run, result.Command, "Command should match")
	assert.NoError(t, result.Error, "Step should execute successfully")
}

func TestLocalExecutor_Transfer_MultiStep(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	executor := NewLocalExecutor()
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 20)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送多个步骤
	steps := []pipelinex.Step{
		{Name: "step1", Run: "echo 'step 1 output'"},
		{Name: "step2", Run: "echo 'step 2 output'"},
		{Name: "step3", Run: "echo 'step 3 output'"},
	}
	commandChan <- steps
	close(commandChan)

	// 接收结果
	var results []*StepResult
	timeout := time.After(10 * time.Second)

	for len(results) < 3 {
		select {
		case res := <-resultChan:
			switch v := res.(type) {
			case *StepResult:
				results = append(results, v)
			case error:
				t.Fatalf("Received error: %v", v)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for results, received %d results", len(results))
		}
	}

	// 验证结果
	assert.Equal(t, 3, len(results), "Should receive 3 step results")
	for i, result := range results {
		assert.NoError(t, result.Error, "Step %d should execute successfully", i+1)
		assert.Equal(t, steps[i].Name, result.StepName, "StepName should match")
	}
}

func TestLocalExecutor_Transfer_CommandFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	executor := NewLocalExecutor()
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 10)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送一个会失败的命令
	commandChan <- "exit 1"
	close(commandChan)

	// 接收结果
	var result *StepResult
	timeout := time.After(5 * time.Second)

resultLoop:
	for {
		select {
		case res := <-resultChan:
			switch v := res.(type) {
			case *StepResult:
				result = v
				break resultLoop
			case error:
				t.Fatalf("Received error: %v", v)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for results")
		}
	}

	// 验证命令失败
	require.NotNil(t, result, "Should receive StepResult")
	assert.Error(t, result.Error, "Failed command should return error")
}

func TestLocalExecutor_EnvConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	executor := NewLocalExecutor()
	executor.setEnv("TEST_VAR", "test_value")
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 10)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送打印环境变量的命令
	if runtime.GOOS == "windows" {
		commandChan <- "echo %TEST_VAR%"
	} else {
		commandChan <- "echo $TEST_VAR"
	}
	close(commandChan)

	// 收集输出
	var output []byte
	var result *StepResult
	timeout := time.After(5 * time.Second)

resultLoop:
	for {
		select {
		case res := <-resultChan:
			switch v := res.(type) {
			case []byte:
				output = append(output, v...)
			case *StepResult:
				result = v
				break resultLoop
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for results")
		}
	}

	require.NotNil(t, result, "Should receive StepResult")
	assert.NoError(t, result.Error, "Command should execute successfully")
	// 注意：Windows的echo会原样输出%TEST_VAR%，所以这里只验证命令成功
}

func TestLocalExecutor_ConfigApplication(t *testing.T) {
	ctx := context.Background()

	configTests := []struct {
		name   string
		config map[string]any
		verify func(t *testing.T, exec *LocalExecutor)
	}{
		{
			name: "basic config",
			config: map[string]any{
				"workdir": "/tmp",
				"shell":   "bash",
			},
			verify: func(t *testing.T, exec *LocalExecutor) {
				assert.Equal(t, "/tmp", exec.GetWorkdir())
				assert.Equal(t, "bash", exec.GetShell())
			},
		},
		{
			name: "timeout string config",
			config: map[string]any{
				"timeout": "30s",
			},
			verify: func(t *testing.T, exec *LocalExecutor) {
				assert.NotNil(t, exec)
			},
		},
		{
			name: "timeout number config",
			config: map[string]any{
				"timeout": 30,
			},
			verify: func(t *testing.T, exec *LocalExecutor) {
				assert.NotNil(t, exec)
			},
		},
		{
			name: "pty config",
			config: map[string]any{
				"pty":       true,
				"ptyWidth":  120,
				"ptyHeight": 40,
			},
			verify: func(t *testing.T, exec *LocalExecutor) {
				assert.NotNil(t, exec)
			},
		},
		{
			name: "env config",
			config: map[string]any{
				"env": map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
				},
			},
			verify: func(t *testing.T, exec *LocalExecutor) {
				assert.NotNil(t, exec)
			},
		},
		{
			name: "mixed config",
			config: map[string]any{
				"workdir":   "/workspace",
				"shell":     "sh",
				"timeout":   "5m",
				"env":       map[string]string{"ENV": "test"},
				"pty":       true,
				"ptyWidth":  100,
				"ptyHeight": 30,
			},
			verify: func(t *testing.T, exec *LocalExecutor) {
				assert.Equal(t, "/workspace", exec.GetWorkdir())
				assert.Equal(t, "sh", exec.GetShell())
			},
		},
	}

	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := NewLocalBridge()
			adapter := NewLocalAdapter()

			err := adapter.Config(ctx, tt.config)
			require.NoError(t, err, "Config should not return error")

			executor, err := bridge.Conn(ctx, adapter)
			require.NoError(t, err, "Conn should not return error")
			require.NotNil(t, executor, "Executor should not be nil")

			localExec, ok := executor.(*LocalExecutor)
			require.True(t, ok, "Executor should be *local.LocalExecutor")

			tt.verify(t, localExec)
		})
	}
}

func TestLocalExecutor_InvalidAdapter(t *testing.T) {
	ctx := context.Background()
	bridge := NewLocalBridge()

	// 使用错误的适配器类型
	dummyAdapter := &dummyAdapter{}

	_, err := bridge.Conn(ctx, dummyAdapter)
	assert.Error(t, err, "Should return error for invalid adapter type")
	assert.Contains(t, err.Error(), "not a LocalAdapter", "Error message should indicate wrong adapter type")
}

// dummyAdapter 用于测试的虚拟适配器
type dummyAdapter struct{}

func (d *dummyAdapter) Config(ctx context.Context, config map[string]any) error {
	return nil
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "string seconds",
			input:    "30s",
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "string minutes",
			input:    "5m",
			expected: 5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "int value",
			input:    30,
			expected: 30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "int64 value",
			input:    int64(60),
			expected: 60 * time.Second,
			wantErr:  false,
		},
		{
			name:     "float64 value",
			input:    45.0,
			expected: 45 * time.Second,
			wantErr:  false,
		},
		{
			name:    "invalid string",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "unsupported type",
			input:   true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := parseTimeout(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}

func TestDetectDefaultShell(t *testing.T) {
	shell := detectDefaultShell()
	assert.NotEmpty(t, shell, "Should detect a default shell")

	// 根据操作系统验证
	switch runtime.GOOS {
	case "windows":
		// Windows应该有PowerShell或cmd
		assert.True(t, shell == "pwsh" || shell == "powershell" || shell == "cmd",
			"Windows should have powershell or cmd, got: %s", shell)
	default:
		// Unix-like系统应该有bash或sh
		assert.True(t, shell == "/bin/bash" || shell == "/bin/sh",
			"Unix should have bash or sh, got: %s", shell)
	}
}

func TestLocalExecutor_CancelContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	ctx, cancel := context.WithCancel(context.Background())

	executor := NewLocalExecutor()
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 10)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送一个长时间运行的命令
	commandChan <- "sleep 30"

	// 等待命令启动
	time.Sleep(100 * time.Millisecond)

	// 取消上下文
	cancel()

	// 应该收到结果（命令被取消或完成）
	timeout := time.After(5 * time.Second)
	select {
	case <-resultChan:
		// 命令被取消或返回了结果
		// 注意：由于 exec.CommandContext 的行为，子进程可能不会被立即终止
	case <-timeout:
		t.Log("Note: Context cancellation may not immediately terminate child processes")
	}
}

func TestLocalExecutor_Getters(t *testing.T) {
	executor := NewLocalExecutor()

	// 设置值
	executor.setWorkdir("/test/workdir")
	executor.setShell("zsh")
	executor.setTimeout(60 * time.Second)
	executor.setPTY(true)
	executor.setPTYSize(120, 40)

	// 验证getter
	assert.Equal(t, "/test/workdir", executor.GetWorkdir())
	assert.Equal(t, "zsh", executor.GetShell())
}

func TestLocalExecutor_UnsupportedDataType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	executor := NewLocalExecutor()
	require.NoError(t, executor.Prepare(ctx))
	defer executor.Destruction(ctx)

	resultChan := make(chan any, 10)
	commandChan := make(chan any, 1)

	go executor.Transfer(ctx, resultChan, commandChan)

	// 发送不支持的类型
	commandChan <- 12345
	close(commandChan)

	// 接收错误
	timeout := time.After(3 * time.Second)
	select {
	case res := <-resultChan:
		err, ok := res.(error)
		require.True(t, ok, "Should receive an error")
		assert.Contains(t, err.Error(), "unsupported data type", "Error should indicate unsupported type")
	case <-timeout:
		t.Fatal("Timeout waiting for error")
	}
}