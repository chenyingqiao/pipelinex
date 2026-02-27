package docker

import (
	"context"
	"fmt"

	"github.com/chenyingqiao/pipelinex"
)

// DockerBridge Docker桥接器实现
type DockerBridge struct{}

// NewDockerBridge 创建新的Docker桥接器
func NewDockerBridge() *DockerBridge {
	return &DockerBridge{}
}

// Conn 连接到Docker环境并创建执行器
// adapter: 适配器，包含Docker配置
// 返回: Docker执行器实例
func (b *DockerBridge) Conn(ctx context.Context, adapter pipelinex.Adapter) (pipelinex.Executor, error) {
	// 创建新的Docker执行器
	executor, err := NewDockerExecutor()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker executor: %w", err)
	}

	// 如果提供了适配器，应用配置
	if adapter != nil {
		dockerAdapter, ok := adapter.(*DockerAdapter)
		if !ok {
			return nil, fmt.Errorf("adapter is not a DockerAdapter")
		}

		// 应用配置到执行器
		if err := applyConfigToExecutor(dockerAdapter.config, executor); err != nil {
			return nil, fmt.Errorf("failed to apply adapter config: %w", err)
		}
	}

	return executor, nil
}

// 确保DockerBridge实现了Bridge接口
var _ pipelinex.Bridge = (*DockerBridge)(nil)

// applyConfigToExecutor 将配置应用到执行器
func applyConfigToExecutor(config map[string]any, executor *DockerExecutor) error {
	// 应用registry配置
	if registry, ok := getString(config, "registry"); ok {
		executor.setRegistry(registry)
	}

	// 应用network配置
	if network, ok := getString(config, "network"); ok {
		executor.setNetwork(network)
	}

	// 应用workdir配置
	if workdir, ok := getString(config, "workdir"); ok {
		executor.setWorkdir(workdir)
	}

	// 应用volumes配置
	if volumes, ok := config["volumes"]; ok {
		if volList, ok := volumes.([]string); ok {
			for _, vol := range volList {
				hostPath, containerPath, err := parseVolume(vol)
				if err != nil {
					return fmt.Errorf("invalid volume format %s: %w", vol, err)
				}
				executor.setVolume(hostPath, containerPath)
			}
		}
		if volList, ok := volumes.([]any); ok {
			for _, vol := range volList {
				if volStr, ok := vol.(string); ok {
					hostPath, containerPath, err := parseVolume(volStr)
					if err != nil {
						return fmt.Errorf("invalid volume format %s: %w", volStr, err)
					}
					executor.setVolume(hostPath, containerPath)
				}
			}
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

	// 应用tty配置
	if tty, ok := config["tty"]; ok {
		if ttyBool, ok := tty.(bool); ok {
			executor.setTTY(ttyBool)
		}
	}

	// 应用ttyWidth配置
	if ttyWidth, ok := config["ttyWidth"]; ok {
		if width, ok := ttyWidth.(int); ok {
			executor.setTTYSize(uint(width), 0) // 只设置宽度，高度使用默认
		}
		if width, ok := ttyWidth.(int64); ok {
			executor.setTTYSize(uint(width), 0)
		}
		if width, ok := ttyWidth.(float64); ok {
			executor.setTTYSize(uint(width), 0)
		}
	}

	// 应用ttyHeight配置
	if ttyHeight, ok := config["ttyHeight"]; ok {
		currentWidth := uint(80) // 默认宽度
		if ttyWidth, ok := config["ttyWidth"]; ok {
			if width, ok := ttyWidth.(int); ok {
				currentWidth = uint(width)
			}
			if width, ok := ttyWidth.(int64); ok {
				currentWidth = uint(width)
			}
			if width, ok := ttyWidth.(float64); ok {
				currentWidth = uint(width)
			}
		}

		if height, ok := ttyHeight.(int); ok {
			executor.setTTYSize(currentWidth, uint(height))
		}
		if height, ok := ttyHeight.(int64); ok {
			executor.setTTYSize(currentWidth, uint(height))
		}
		if height, ok := ttyHeight.(float64); ok {
			executor.setTTYSize(currentWidth, uint(height))
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
