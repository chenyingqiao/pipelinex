package pipelinex

import (
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
)

type DGANode struct {
	id         string
	state      string
	property   map[string]any
	pipelineId string
	executor   string
	steps      []Step
	image      string
	config     map[string]any
}

// NewDGANode creates a new DGANode with the specified id and state, initializing an empty property map.
func NewDGANode(id, state string) *DGANode {
	return &DGANode{
		id:       id,
		state:    state,
		property: map[string]any{},
		steps:    []Step{},
		config:   map[string]any{},
	}
}

// NewDGANodeWithConfig creates a new DGANode with full configuration.
func NewDGANodeWithConfig(id, state, executor, image string, steps []Step, config map[string]any) *DGANode {
	return &DGANode{
		id:       id,
		state:    state,
		property: map[string]any{},
		executor: executor,
		steps:    steps,
		image:    image,
		config:   config,
	}
}

func (dgaNode *DGANode) Id() string {
	return dgaNode.id
}

func (dgaNode *DGANode) PipelineId() string {
	return dgaNode.pipelineId
}

func (dgaNode *DGANode) Status() string {
	return dgaNode.state
}

func (dgaNode *DGANode) Get(key string) string {
	return cast.ToString(funk.Get(dgaNode.property, key))
}

func (dgaNode *DGANode) Set(key string, value any) {
	dgaNode.property[key] = value
}

func (dgaNode *DGANode) GetExecutor() string {
	return dgaNode.executor
}

func (dgaNode *DGANode) GetSteps() []Step {
	return dgaNode.steps
}

func (dgaNode *DGANode) GetImage() string {
	return dgaNode.image
}

func (dgaNode *DGANode) GetConfig() map[string]any {
	return dgaNode.config
}
