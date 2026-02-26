package docker

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chenyingqiao/pipelinex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerAdapter_Config(t *testing.T) {
	adapter := NewDockerAdapter()
	ctx := context.Background()

	config := map[string]any{
		"registry": "myregistry.com",
		"network":  "host",
		"workdir":  "/app",
		"volumes": []string{
			"/var/run/docker.sock:/var/run/docker.sock",
		},
		"env": map[string]string{
			"GO_VERSION": "1.21",
		},
		"tty":       true,
		"ttyWidth":  120,
		"ttyHeight": 40,
	}

	err := adapter.Config(ctx, config)
	if err != nil {
		t.Errorf("Config() error = %v", err)
	}
}

func TestDockerBridge_Conn(t *testing.T) {
	bridge := NewDockerBridge()
	ctx := context.Background()

	config := map[string]any{
		"network": "bridge",
		"workdir": "/workspace",
		"tty":     true,
	}

	adapter := NewDockerAdapter()
	err := adapter.Config(ctx, config)
	if err != nil {
		t.Errorf("Config() error = %v", err)
	}

	executor, err := bridge.Conn(ctx, adapter)
	if err != nil {
		t.Errorf("Conn() error = %v", err)
	}

	if executor == nil {
		t.Error("Expected non-nil executor")
	}

	// 验证返回的是 DockerExecutor 类型
	_, ok := executor.(*DockerExecutor)
	if !ok {
		t.Error("Expected executor to be *docker.DockerExecutor")
	}
}

func TestDockerExecutor_Setters(t *testing.T) {
	executor, err := NewDockerExecutor()
	if err != nil {
		t.Fatalf("NewDockerExecutor() error = %v", err)
	}

	// 测试设置镜像
	executor.setImage("golang:1.21-alpine")

	// 测试设置工作目录
	executor.setWorkdir("/app")

	// 测试设置环境变量
	executor.setEnv("KEY1", "value1")
	executor.setEnv("KEY2", "value2")

	// 测试设置卷挂载
	executor.setVolume("/host/path", "/container/path")

	// 测试设置网络
	executor.setNetwork("host")

	// 测试设置仓库
	executor.setRegistry("myregistry.com")

	// 测试 TTY 设置
	executor.setTTY(true)
	executor.setTTYSize(120, 40)

	// 这些方法不应该 panic
	assert.NotPanics(t, func() {
		executor.setTTY(false)
		executor.setTTYSize(80, 24)
	})
}

func TestParseVolume(t *testing.T) {
	tests := []struct {
		name          string
		volume        string
		wantHost      string
		wantContainer string
		wantErr       bool
	}{
		{
			name:          "valid volume",
			volume:        "/host/path:/container/path",
			wantHost:      "/host/path",
			wantContainer: "/container/path",
			wantErr:       false,
		},
		{
			name:          "valid volume with mode",
			volume:        "/host/path:/container/path:ro",
			wantHost:      "/host/path",
			wantContainer: "/container/path",
			wantErr:       false,
		},
		{
			name:    "invalid volume - no colon",
			volume:  "invalidvolume",
			wantErr: true,
		},
		{
			name:    "invalid volume - empty host",
			volume:  ":/container/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, container, err := parseVolume(tt.volume)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if host != tt.wantHost {
					t.Errorf("parseVolume() host = %v, want %v", host, tt.wantHost)
				}
				if container != tt.wantContainer {
					t.Errorf("parseVolume() container = %v, want %v", container, tt.wantContainer)
				}
			}
		})
	}
}

func TestDockerExecutor_Interface(t *testing.T) {
	// 验证 DockerExecutor 实现了 Executor 接口
	var _ pipelinex.Executor = (*DockerExecutor)(nil)
}

func TestDockerAdapter_Interface(t *testing.T) {
	// 验证 DockerAdapter 实现了 Adapter 接口
	var _ pipelinex.Adapter = (*DockerAdapter)(nil)
}

func TestDockerBridge_Interface(t *testing.T) {
	// 验证 DockerBridge 实现了 Bridge 接口
	var _ pipelinex.Bridge = (*DockerBridge)(nil)
}

func TestDockerExecutor_PrepareAndDestruction(t *testing.T) {
	// 注意：此测试需要Docker环境
	// 在没有Docker环境的CI中应该跳过
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 检查Docker是否可用
	if !isDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	executor, err := NewDockerExecutor()
	if err != nil {
		t.Fatalf("NewDockerExecutor() error = %v", err)
	}
	executor.setImage("alpine:latest")

	// 准备环境
	err = executor.Prepare(ctx)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	// 验证容器ID已设置
	containerID := executor.GetContainerID()
	assert.NotEmpty(t, containerID, "Expected non-empty container ID after Prepare")

	// 销毁环境
	err = executor.Destruction(ctx)
	if err != nil {
		t.Errorf("Destruction() error = %v", err)
	}
}

// TestDockerExecutor_ConfigApplication 测试配置应用
func TestDockerExecutor_ConfigApplication(t *testing.T) {
	ctx := context.Background()

	configTests := []struct {
		name   string
		config map[string]any
		verify func(t *testing.T, exec *DockerExecutor)
	}{
		{
			name: "basic config",
			config: map[string]any{
				"registry": "myregistry.com",
				"network":  "host",
				"workdir":  "/app",
			},
			verify: func(t *testing.T, exec *DockerExecutor) {
				// 配置应该被应用，没有错误
				assert.NotNil(t, exec)
			},
		},
		{
			name: "volumes config",
			config: map[string]any{
				"volumes": []string{
					"/host/path1:/container/path1",
					"/host/path2:/container/path2:ro",
				},
			},
			verify: func(t *testing.T, exec *DockerExecutor) {
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
			verify: func(t *testing.T, exec *DockerExecutor) {
				assert.NotNil(t, exec)
			},
		},
		{
			name: "tty config",
			config: map[string]any{
				"tty":       true,
				"ttyWidth":  120,
				"ttyHeight": 40,
			},
			verify: func(t *testing.T, exec *DockerExecutor) {
				assert.NotNil(t, exec)
			},
		},
		{
			name: "mixed config",
			config: map[string]any{
				"registry": "registry.example.com",
				"network":  "bridge",
				"workdir":  "/workspace",
				"volumes":  []string{"/tmp:/tmp"},
				"env":      map[string]string{"ENV": "test"},
				"tty":      true,
			},
			verify: func(t *testing.T, exec *DockerExecutor) {
				assert.NotNil(t, exec)
			},
		},
	}

	for _, tt := range configTests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := NewDockerBridge()
			adapter := NewDockerAdapter()

			err := adapter.Config(ctx, tt.config)
			require.NoError(t, err, "Config should not return error")

			executor, err := bridge.Conn(ctx, adapter)
			require.NoError(t, err, "Conn should not return error")
			require.NotNil(t, executor, "Executor should not be nil")

			dockerExec, ok := executor.(*DockerExecutor)
			require.True(t, ok, "Executor should be *docker.DockerExecutor")

			tt.verify(t, dockerExec)
		})
	}
}

// TestDockerExecutor_AdapterConfigWithInvalidTypes 测试无效配置类型的处理
func TestDockerExecutor_AdapterConfigWithInvalidTypes(t *testing.T) {
	ctx := context.Background()

	invalidConfigTests := []struct {
		name   string
		config map[string]any
	}{
		{
			name: "invalid tty type",
			config: map[string]any{
				"tty": "true", // 应该是 bool，不是 string
			},
		},
		{
			name: "invalid ttyWidth type",
			config: map[string]any{
				"ttyWidth": "120", // 应该是数字，不是 string
			},
		},
		{
			name: "invalid volumes format",
			config: map[string]any{
				"volumes": "not-a-list", // 应该是 []string
			},
		},
		{
			name: "invalid env format",
			config: map[string]any{
				"env": "not-a-map", // 应该是 map
			},
		},
	}

	for _, tt := range invalidConfigTests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := NewDockerBridge()
			adapter := NewDockerAdapter()

			// 配置不应该返回错误（只是忽略无效的配置）
			err := adapter.Config(ctx, tt.config)
			// 我们期望配置被接受，但无效值会被忽略
			assert.NoError(t, err, "Config should accept the config even with invalid types")

			// 连接应该成功
			executor, err := bridge.Conn(ctx, adapter)
			assert.NoError(t, err, "Conn should succeed")
			assert.NotNil(t, executor, "Executor should not be nil")
		})
	}
}

// TestDockerExecutor_VolumeParsing 测试卷挂载解析的各种格式
func TestDockerExecutor_VolumeParsing(t *testing.T) {
	ctx := context.Background()

	volumeTests := []struct {
		name        string
		volumes     any
		expectError bool
		volumeCount int
	}{
		{
			name: "string list volumes",
			volumes: []string{
				"/host1:/container1",
				"/host2:/container2:ro",
			},
			expectError: false,
			volumeCount: 2,
		},
		{
			name: "any list volumes",
			volumes: []any{
				"/host1:/container1",
				"/host2:/container2:ro",
			},
			expectError: false,
			volumeCount: 2,
		},
		{
			name:        "no volumes",
			volumes:     nil,
			expectError: false,
			volumeCount: 0,
		},
		{
			name:        "empty list",
			volumes:     []string{},
			expectError: false,
			volumeCount: 0,
		},
	}

	for _, tt := range volumeTests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := NewDockerBridge()
			adapter := NewDockerAdapter()

			config := map[string]any{
				"volumes": tt.volumes,
			}

			err := adapter.Config(ctx, config)
			assert.NoError(t, err, "Config should not return error")

			executor, err := bridge.Conn(ctx, adapter)
			if tt.expectError {
				assert.Error(t, err, "Expected error for invalid volumes")
			} else {
				assert.NoError(t, err, "Conn should not return error")
				assert.NotNil(t, executor, "Executor should not be nil")
			}
		})
	}
}

// TestDockerExecutor_EnvConfigParsing 测试环境变量配置的各种格式
func TestDockerExecutor_EnvConfigParsing(t *testing.T) {
	ctx := context.Background()

	envTests := []struct {
		name        string
		env         any
		expectError bool
		envCount    int
	}{
		{
			name: "map[string]string env",
			env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			expectError: false,
			envCount:    2,
		},
		{
			name: "map[string]any env",
			env: map[string]any{
				"KEY1": "value1",
				"KEY2": 123, // 会被忽略
			},
			expectError: false,
			envCount:    1, // 只有字符串值会被应用
		},
		{
			name:        "no env",
			env:         nil,
			expectError: false,
			envCount:    0,
		},
		{
			name:        "empty map",
			env:         map[string]string{},
			expectError: false,
			envCount:    0,
		},
	}

	for _, tt := range envTests {
		t.Run(tt.name, func(t *testing.T) {
			bridge := NewDockerBridge()
			adapter := NewDockerAdapter()

			config := map[string]any{
				"env": tt.env,
			}

			err := adapter.Config(ctx, config)
			assert.NoError(t, err, "Config should not return error")

			executor, err := bridge.Conn(ctx, adapter)
			if tt.expectError {
				assert.Error(t, err, "Expected error for invalid env")
			} else {
				assert.NoError(t, err, "Conn should not return error")
				assert.NotNil(t, executor, "Executor should not be nil")
			}
		})
	}
}

// isDockerAvailable 检查Docker是否可用
func isDockerAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "version")
	err := cmd.Run()
	return err == nil
}

// TestDockerExecutor_IntegrationWithDocker 集成测试 - 需要真实的Docker环境
func TestDockerExecutor_IntegrationWithDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if !isDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping integration test")
	}

	t.Run("full workflow", func(t *testing.T) {
		executor, err := NewDockerExecutor()
		require.NoError(t, err)

		// 配置执行器
		executor.setImage("alpine:latest")
		executor.setTTY(true)
		executor.setTTYSize(80, 24)

		// 准备环境
		err = executor.Prepare(ctx)
		require.NoError(t, err, "Prepare should succeed")

		containerID := executor.GetContainerID()
		assert.NotEmpty(t, containerID, "Container ID should be set")

		// 清理环境
		defer func() {
			err := executor.Destruction(ctx)
			assert.NoError(t, err, "Destruction should succeed")
		}()

		// 测试 Transfer 方法
		resultChan := make(chan any, 10)
		commandChan := make(chan any, 1)

		go executor.Transfer(ctx, resultChan, commandChan)

		// 发送测试命令
		testCommand := "echo 'Hello from Docker'"
		commandChan <- testCommand

		// 接收结果
		results := []any{}
		timeout := time.After(5 * time.Second)
		resultCount := 0

		for resultCount < 2 { // 预期：输出 + 结果
			select {
			case result := <-resultChan:
				results = append(results, result)
				resultCount++

				// 验证输出内容
				if data, ok := result.([]byte); ok {
					output := string(data)
					t.Logf("Received output: %q", output)
					// 验证输出包含我们的测试字符串
					if strings.Contains(output, "Hello from Docker") {
						t.Log("✓ Received expected output")
					}
				}

				if stepResult, ok := result.(*StepResult); ok {
					t.Logf("Received step result: command=%q, error=%v", stepResult.Command, stepResult.Error)
					assert.Equal(t, testCommand, stepResult.Command)
					assert.NoError(t, stepResult.Error, "Command should execute successfully")
				}

			case <-timeout:
				t.Fatalf("Timeout waiting for results, received %d results", resultCount)
			}
		}

		assert.GreaterOrEqual(t, len(results), 1, "Should receive at least one result")
	})

	t.Run("TTY mode test", func(t *testing.T) {
		executor, err := NewDockerExecutor()
		require.NoError(t, err)

		executor.setImage("alpine:latest")
		executor.setTTY(true) // 启用 TTY

		err = executor.Prepare(ctx)
		require.NoError(t, err)

		defer executor.Destruction(ctx)

		resultChan := make(chan any, 10)
		commandChan := make(chan any, 1)

		go executor.Transfer(ctx, resultChan, commandChan)

		// 发送会产生颜色输出的命令
		commandChan <- "ls --color=always /"

		// 接收结果
		select {
		case result := <-resultChan:
			if data, ok := result.([]byte); ok {
				t.Logf("TTY output length: %d bytes", len(data))
				// TTY 模式下应该保留 ANSI 转义序列
				t.Logf("Output preview: %q", string(data[:min(100, len(data))]))
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for TTY output")
		}
	})

	t.Run("multi-step execution", func(t *testing.T) {
		executor, err := NewDockerExecutor()
		require.NoError(t, err)

		executor.setImage("alpine:latest")
		executor.setTTY(false)

		err = executor.Prepare(ctx)
		require.NoError(t, err)

		defer executor.Destruction(ctx)

		resultChan := make(chan any, 20)
		commandChan := make(chan any, 1)

		go executor.Transfer(ctx, resultChan, commandChan)

		// 发送多个步骤
		steps := []pipelinex.Step{
			{Name: "echo", Run: "echo 'step 1'"},
			{Name: "pwd", Run: "pwd"},
			{Name: "ls", Run: "ls /"},
		}
		commandChan <- steps

		// 接收结果（每个步骤：输出 + 结果 = 6个消息）
		results := []any{}
		timeout := time.After(10 * time.Second)

		for len(results) < 6 {
			select {
			case result := <-resultChan:
				results = append(results, result)
				t.Logf("Received result %d: %T", len(results), result)
			case <-timeout:
				t.Fatalf("Timeout, received %d results", len(results))
			}
		}

		// 验证收到所有步骤的结果
		stepResultCount := 0
		for _, result := range results {
			if _, ok := result.(*StepResult); ok {
				stepResultCount++
			}
		}
		assert.Equal(t, 3, stepResultCount, "Should receive 3 step results")
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
