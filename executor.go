package pipelinex

import "context"

// Executor 执行器
type Executor interface {
	// Prepare 准备环境
	Prepare(ctx context.Context) error
	// Destruction 销毁环境
	Destruction(ctx context.Context) error
}

type Adapter interface{
	// Config 适配器配置
	Config(config map[string]interface{}) Adapter
	// Conn 连接到环境中
	Conn(ctx context.Context) interface{}
}

type Bridge interface{
	// Transfer 传输需要执行的数据，并且反回执行的结果
	Transfer(ctx context.Context, in chan<- interface{}, out <-chan interface{}) 
}
