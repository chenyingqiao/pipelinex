package pipelinex

import "context"

var (
	//监听事件
	PipelineInit                Event = "pipeline-init"  // 流水线初始化
	PipelineStart               Event = "pipeline-start" // 流水线开始执行
	PipelineFinish              Event = "pipeline-finish"
	PipelineExecutorPrepare     Event = "pipeline-executor-prepare"      // 流水线执行器开始准备
	PipelineExecutorPrepareDone Event = "pipeline-executor-prepare-done" // 流水线执行器准备完毕
	PipelineNodeStart           Event = "pipeline-node-start"
	PipelineNodeFinish          Event = "pipeline-node-finish"
)

type TraversalFn func(ctx context.Context, node Node) error

type Graph interface {
	GraphReader
	//AddVertex 添加顶点
	AddVertex(node Node)
	//AddEdge 添加边
	AddEdge(src, dest Node) error
}

type GraphReader interface {
	//Nodes
	Nodes() map[string]Node
	//Traversal 遍历图结构
	Traversal(ctx context.Context, fn TraversalFn) error
}

// 流水线事件
type Event string

// 我们将整个流水线的运行过程中的事件抽象成对应的Event
// 这样我们就能再外部监听Event
type Listener interface {
	// 处理对应的事件将事件发生的对应的流水线和对应的事件作为参数传入
	Handle(p Pipeline, event Event)
	// 获取当前注册的Event
	Events() []Event
}

// PipelineListeningFn 流水线监听函数
type ListeningFn func(p Pipeline)
type Metadata map[string]any

type Pipeline interface {
	//ID 流水线的id
	Id() string
	//GetGraph 返回图结构
	GetGraph() Graph
	//SetGraph 设置图结构
	SetGraph(graph Graph)
	//Status 返回流水线的整体状态
	Status() string
	//Metadata 返回流水线执行的源数据
	Metadata() Metadata
	//Listening 流水线执行事件监听设置
	Listening(listener Listener)
	//Done流水线是否执行完成
	Done() <-chan struct{}
	//Run执行流水线
	Run(ctx context.Context) error
	//Notify 执行的步骤通知流水线
	Notify()
	//Cancel 取消流水线
	Cancel()
}
