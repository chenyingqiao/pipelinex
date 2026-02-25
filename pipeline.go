package pipelinex

import "context"

var (
	//监听事件
	PipelineInit                Event = EventPipelineInit                // 流水线初始化
	PipelineStart               Event = EventPipelineStart               // 流水线开始执行
	PipelineFinish              Event = EventPipelineFinish              // 流水线完成
	PipelineExecutorPrepare     Event = EventPipelineExecutorPrepare     // 流水线执行器开始准备
	PipelineExecutorPrepareDone Event = EventPipelineExecutorPrepareDone // 流水线执行器准备完毕
	PipelineNodeStart           Event = EventPipelineNodeStart           // 节点开始
	PipelineNodeFinish          Event = EventPipelineNodeFinish          // 节点完成
)

type TraversalFn func(ctx context.Context, node Node) error

type Graph interface {
	GraphReader
	//AddVertex 添加顶点
	AddVertex(node Node)
	//AddEdge 添加边
	AddEdge(edge Edge) error
}

type GraphReader interface {
	//Nodes
	Nodes() map[string]Node
	//Edges 返回所有的边
	Edges() []Edge
	//Traversal 遍历图结构
	Traversal(ctx context.Context, evalCtx EvaluationContext, fn TraversalFn) error
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
	//SetMetadata 设置元数据
	SetMetadata(store MetadataStore)
	//Metadata 获取元数据
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
