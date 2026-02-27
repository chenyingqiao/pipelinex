package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chenyingqiao/pipelinex"
)

// KubernetesBridge Kubernetes桥接器实现
type KubernetesBridge struct{}

// NewKubernetesBridge 创建新的Kubernetes桥接器
func NewKubernetesBridge() *KubernetesBridge {
	return &KubernetesBridge{}
}

// Conn 连接到Kubernetes环境并创建执行器
// adapter: 适配器，包含Kubernetes配置
// 返回: Kubernetes执行器实例
func (b *KubernetesBridge) Conn(ctx context.Context, adapter pipelinex.Adapter) (pipelinex.Executor, error) {
	// 创建新的Kubernetes执行器
	executor, err := NewKubernetesExecutor()
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes executor: %w", err)
	}

	// 如果提供了适配器，应用配置
	if adapter != nil {
		k8sAdapter, ok := adapter.(*KubernetesAdapter)
		if !ok {
			return nil, fmt.Errorf("adapter is not a KubernetesAdapter")
		}

		// 应用配置到执行器
		if err := applyConfigToExecutor(k8sAdapter.config, executor); err != nil {
			return nil, fmt.Errorf("failed to apply adapter config: %w", err)
		}
	}

	return executor, nil
}

// 确保KubernetesBridge实现了Bridge接口
var _ pipelinex.Bridge = (*KubernetesBridge)(nil)

// applyConfigToExecutor 将配置应用到执行器
func applyConfigToExecutor(config map[string]any, executor *KubernetesExecutor) error {
	// 应用namespace配置
	if namespace, ok := getString(config, "namespace"); ok {
		executor.setNamespace(namespace)
	}

	// 应用image配置
	if image, ok := getString(config, "image"); ok {
		executor.setImage(image)
	}

	// 应用workdir配置
	if workdir, ok := getString(config, "workdir"); ok {
		executor.setWorkdir(workdir)
	}

	// 应用serviceAccount配置
	if sa, ok := getString(config, "serviceAccount"); ok {
		executor.setServiceAccount(sa)
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

	// 应用configMaps配置
	if configMaps, ok := config["configMaps"]; ok {
		if cmList, ok := configMaps.([]map[string]any); ok {
			for _, cm := range cmList {
				vol, mount, err := parseConfigMapVolume(cm)
				if err != nil {
					return fmt.Errorf("invalid configmap volume: %w", err)
				}
				executor.setVolume(vol, mount)
			}
		}
		if cmList, ok := configMaps.([]any); ok {
			for _, cm := range cmList {
				if cmMap, ok := cm.(map[string]any); ok {
					vol, mount, err := parseConfigMapVolume(cmMap)
					if err != nil {
						return fmt.Errorf("invalid configmap volume: %w", err)
					}
					executor.setVolume(vol, mount)
				}
			}
		}
	}

	// 应用secrets配置
	if secrets, ok := config["secrets"]; ok {
		if secretList, ok := secrets.([]map[string]any); ok {
			for _, secret := range secretList {
				vol, mount, err := parseSecretVolume(secret)
				if err != nil {
					return fmt.Errorf("invalid secret volume: %w", err)
				}
				executor.setVolume(vol, mount)
			}
		}
		if secretList, ok := secrets.([]any); ok {
			for _, secret := range secretList {
				if secretMap, ok := secret.(map[string]any); ok {
					vol, mount, err := parseSecretVolume(secretMap)
					if err != nil {
						return fmt.Errorf("invalid secret volume: %w", err)
					}
					executor.setVolume(vol, mount)
				}
			}
		}
	}

	// 应用volumes (PVC) 配置
	if volumes, ok := config["volumes"]; ok {
		if volList, ok := volumes.([]map[string]any); ok {
			for _, vol := range volList {
				volume, mount, err := parsePVCVolume(vol)
				if err != nil {
					return fmt.Errorf("invalid pvc volume: %w", err)
				}
				executor.setVolume(volume, mount)
			}
		}
		if volList, ok := volumes.([]any); ok {
			for _, vol := range volList {
				if volMap, ok := vol.(map[string]any); ok {
					volume, mount, err := parsePVCVolume(volMap)
					if err != nil {
						return fmt.Errorf("invalid pvc volume: %w", err)
					}
					executor.setVolume(volume, mount)
				}
			}
		}
	}

	// 应用emptyDirs配置
	if emptyDirs, ok := config["emptyDirs"]; ok {
		if volList, ok := emptyDirs.([]map[string]any); ok {
			for _, vol := range volList {
				volume, mount, err := parseEmptyDirVolume(vol)
				if err != nil {
					return fmt.Errorf("invalid emptydir volume: %w", err)
				}
				executor.setVolume(volume, mount)
			}
		}
		if volList, ok := emptyDirs.([]any); ok {
			for _, vol := range volList {
				if volMap, ok := vol.(map[string]any); ok {
					volume, mount, err := parseEmptyDirVolume(volMap)
					if err != nil {
						return fmt.Errorf("invalid emptydir volume: %w", err)
					}
					executor.setVolume(volume, mount)
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

	// 应用podReadyTimeout配置（Pod就绪等待超时时间，单位：秒）
	if timeout, ok := config["podReadyTimeout"]; ok {
		var timeoutSec int
		switch v := timeout.(type) {
		case int:
			timeoutSec = v
		case int64:
			timeoutSec = int(v)
		case float64:
			timeoutSec = int(v)
		case string:
			// 尝试解析字符串为整数
			if parsed, err := parseDurationString(v); err == nil {
				timeoutSec = int(parsed.Seconds())
			}
		}
		if timeoutSec > 0 {
			executor.setPodReadyTimeout(time.Duration(timeoutSec) * time.Second)
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

// parseDurationString 解析时长字符串（支持纯数字或带单位，如 "60s", "2m", "120"）
func parseDurationString(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// 尝试直接解析（支持 Go 的 duration 格式，如 "60s", "2m30s"）
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// 尝试解析为纯数字（默认单位为秒）
	if sec, err := strconv.Atoi(s); err == nil {
		return time.Duration(sec) * time.Second, nil
	}

	return 0, fmt.Errorf("invalid duration string: %s", s)
}
