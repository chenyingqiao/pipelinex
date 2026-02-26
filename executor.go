package pipelinex

import "context"

// Executor 执行器
type Executor interface {
	// Prepare 准备环境
	Prepare(ctx context.Context) error
	// Destruction 销毁环境
	Destruction(ctx context.Context) error
	// Transfer 从 commandChan 接收命令执行，并将结果发送到 resultChan
	Transfer(ctx context.Context, resultChan chan<- any, commandChan <-chan any)
}

type Adapter interface {
	// Config 适配器配置
	Config(ctx context.Context, config map[string]any) error
}

type Bridge interface {
	// Conn 连接到环境中
	Conn(ctx context.Context, adapter Adapter) (Executor, error)
}
