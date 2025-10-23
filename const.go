package pipelinex

const (
	// 流水线状态常量
	StatusRunning   = "RUNNING"
	StatusFailed    = "FAILED"
	StatusSuccess   = "SUCCESS"
	StatusTerminate = "ABORTED"
	StatusPaused    = "PAUSED"
	StatusUnknown   = "UNKNOWN"
	StatusCancelled = "CANCELLED"

	// 流水线事件常量
	EventPipelineInit                = "pipeline-init"
	EventPipelineStart               = "pipeline-start"
	EventPipelineFinish              = "pipeline-finish"
	EventPipelineExecutorPrepare     = "pipeline-executor-prepare"
	EventPipelineExecutorPrepareDone = "pipeline-executor-prepare-done"
	EventPipelineNodeStart           = "pipeline-node-start"
	EventPipelineNodeFinish          = "pipeline-node-finish"
	EventPipelineCancelled           = "pipeline-cancelled"
	EventPipelineStatusUpdate        = "pipeline-status-update"
)
