package local

import (
	"context"
	"fmt"

	"github.com/chenyingqiao/pipelinex/executor"
)

// LocalBridge 本地桥接器实现
type LocalBridge struct{}

// NewLocalBridge 创建新的本地桥接器
func NewLocalBridge() *LocalBridge {
	return &LocalBridge{}
}

// Conn 连接到本地环境并创建执行器
// adapter: 适配器，包含本地执行配置
// 返回: 本地执行器实例
func (b *LocalBridge) Conn(ctx context.Context, adapter executor.Adapter) (executor.Executor, error) {
	// 创建新的本地执行器
	executor := NewLocalExecutor()

	// 如果提供了适配器，应用配置
	if adapter != nil {
		localAdapter, ok := adapter.(*LocalAdapter)
		if !ok {
			return nil, fmt.Errorf("adapter is not a LocalAdapter")
		}

		// 应用配置到执行器
		if err := applyConfigToExecutor(localAdapter.config, executor); err != nil {
			return nil, fmt.Errorf("failed to apply adapter config: %w", err)
		}
	}

	return executor, nil
}

// 确保LocalBridge实现了Bridge接口
var _ executor.Bridge = (*LocalBridge)(nil)

// applyConfigToExecutor 将配置应用到执行器
func applyConfigToExecutor(config map[string]any, executor *LocalExecutor) error {
	// 应用workdir配置
	if workdir, ok := getString(config, "workdir"); ok {
		executor.setWorkdir(workdir)
	}

	// 应用shell配置
	if shell, ok := getString(config, "shell"); ok {
		executor.setShell(shell)
	}

	// 应用timeout配置
	if timeout, ok := config["timeout"]; ok {
		if duration, err := parseTimeout(timeout); err == nil {
			executor.setTimeout(duration)
		}
	}

	// 应用env配置
	if env, ok := config["env"]; ok {
		if envMap, ok := env.(map[string]string); ok {
			for k, v := range envMap {
				executor.setEnv(k, v)
			}
		}
		if envMap, ok := env.(map[string]any); ok {
			for k, v := range envMap {
				if vStr, ok := v.(string); ok {
					executor.setEnv(k, vStr)
				}
			}
		}
	}

	// 应用pty配置
	if pty, ok := config["pty"]; ok {
		if ptyBool, ok := pty.(bool); ok {
			executor.setPTY(ptyBool)
		}
	}

	// 应用ptyWidth配置
	if ptyWidth, ok := config["ptyWidth"]; ok {
		if width, ok := ptyWidth.(int); ok {
			executor.setPTYSize(width, 0) // 只设置宽度，高度使用默认
		}
		if width, ok := ptyWidth.(int64); ok {
			executor.setPTYSize(int(width), 0)
		}
		if width, ok := ptyWidth.(float64); ok {
			executor.setPTYSize(int(width), 0)
		}
	}

	// 应用ptyHeight配置
	if ptyHeight, ok := config["ptyHeight"]; ok {
		currentWidth := 80 // 默认宽度
		if ptyWidth, ok := config["ptyWidth"]; ok {
			if width, ok := ptyWidth.(int); ok {
				currentWidth = width
			}
			if width, ok := ptyWidth.(int64); ok {
				currentWidth = int(width)
			}
			if width, ok := ptyWidth.(float64); ok {
				currentWidth = int(width)
			}
		}

		if height, ok := ptyHeight.(int); ok {
			executor.setPTYSize(currentWidth, height)
		}
		if height, ok := ptyHeight.(int64); ok {
			executor.setPTYSize(currentWidth, int(height))
		}
		if height, ok := ptyHeight.(float64); ok {
			executor.setPTYSize(currentWidth, int(height))
		}
	}

	return nil
}

// getString 从map中获取字符串值
func getString(m map[string]any, key string) (string, bool) {
	val, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}