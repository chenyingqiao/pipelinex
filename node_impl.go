package pipelinex

import (
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
)

type DGANode struct {
	property map[string]interface{}
}

func NewDGANode(id, groupId, status string) *DGANode {
	return &DGANode{
		property: map[string]interface{}{
			"Id":      id,
			"groupId": groupId,
			"status":  status,
		},
	}
}

func (dgaNode *DGANode) Id() string {
	return cast.ToString(funk.Get(dgaNode.property, "Id"))
}

func (dgaNode *DGANode) GroupId() string {
	return cast.ToString(funk.Get(dgaNode.property, "GroupId"))
}

func (dgaNode *DGANode) Status() string {
	return cast.ToString(funk.Get(dgaNode.property, "Status"))
}

func (dgaNode *DGANode) Get(key string) string {
	return cast.ToString(funk.Get(dgaNode.property, key))
}

func (dgaNode *DGANode) Set(key string, value interface{}) {
	dgaNode.property[key] = value
}
