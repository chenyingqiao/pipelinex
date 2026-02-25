package pipelinex

import "context"

// Runtime 运行时
type Runtime interface {
	//获取流水线状态
	Get(id string) (Pipeline, error)
	//取消运行中的流水线
	Cancel(ctx context.Context, id string) error
	//执行异步流水线
	RunAsync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error)
	//执行同步流水线
	RunSync(ctx context.Context, id string, config string, listener Listener) (Pipeline, error)
	//移除流水线记录
	Rm(id string)
	//runtime已经执行完成
	Done() chan struct{}
	//通知runtime
	Notify(data interface{}) error
	//反回runtime公共
	Ctx() context.Context
	//停止后台处理
	StopBackground()
	// 启动后台
	StartBackground()
	// 设置日志推送器
	SetPusher(pusher Pusher)
	// 设置模板引擎
	SetTemplateEngine(engine TemplateEngine)
}
