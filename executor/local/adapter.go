package local

import (
	"context"
	"fmt"
	"time"

	"github.com/chenyingqiao/pipelinex"
)

// LocalAdapter 本地适配器实现
type LocalAdapter struct {
	config map[string]any
}

// NewLocalAdapter 创建新的本地适配器
func NewLocalAdapter() *LocalAdapter {
	return &LocalAdapter{
		config: make(map[string]any),
	}
}

// Config 配置适配器
// 支持的配置项：
//   - workdir: 工作目录 string
//   - env: 环境变量 map[string]string
//   - shell: 指定shell string (例如: "bash", "sh", "powershell", "cmd")
//   - timeout: 默认超时时间 string (例如: "30s", "5m")
//   - pty: 是否启用伪终端 bool
//   - ptyWidth: 终端宽度 int（默认 80）
//   - ptyHeight: 终端高度 int（默认 24）
func (a *LocalAdapter) Config(ctx context.Context, config map[string]any) error {
	a.config = config
	return nil
}

// 确保LocalAdapter实现了Adapter接口
var _ pipelinex.Adapter = (*LocalAdapter)(nil)

// parseTimeout 解析超时时间配置
func parseTimeout(timeout any) (time.Duration, error) {
	switch v := timeout.(type) {
	case string:
		return time.ParseDuration(v)
	case int:
		return time.Duration(v) * time.Second, nil
	case int64:
		return time.Duration(v) * time.Second, nil
	case float64:
		return time.Duration(v) * time.Second, nil
	default:
		return 0, fmt.Errorf("unsupported timeout type: %T", timeout)
	}
}