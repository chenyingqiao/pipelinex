package test

import (
	"testing"

	"github.com/chenyingqiao/pipelinex"
)

func TestDGA_BFS(t *testing.T) {
	dgaGraph := pipelinex.NewDGAGraph()
	node1 := pipelinex.NewDGANode("a", "RUNNING")
	node2 := pipelinex.NewDGANode("b", "UNKONEW")
	node3 := pipelinex.NewDGANode("c", "UNKONEW")
	node4 := pipelinex.NewDGANode("e", "UNKONEW")
	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)
	dgaGraph.AddVertex(node4)
	dgaGraph.AddEdge(node1, node2)
	dgaGraph.AddEdge(node1, node4)
	dgaGraph.AddEdge(node2, node3)
	dgaGraph.AddEdge(node4, node3)
	dgaGraph.Traversal(func(node pipelinex.Node) {
		println(node.Id())
	})
}
