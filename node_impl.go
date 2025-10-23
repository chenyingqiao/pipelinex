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
}

// NewDGANode creates a new DGANode with the specified id and state, initializing an empty property map.
func NewDGANode(id, state string) *DGANode {
	return &DGANode{
		id:       id,
		state:    state,
		property: map[string]any{},
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
