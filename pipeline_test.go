package pipelinex

import (
	"testing"
)

func TestDGA_BFS(t *testing.T) {
	dgaGraph := NewDGAGraph()
	node1 := NewDGANode("a", "", "RUNNING")
	node2 := NewDGANode("b", "", "UNKONEW")
	node3 := NewDGANode("c", "", "UNKONEW")
	node4 := NewDGANode("e", "", "UNKONEW")
	dgaGraph.AddVertex(node1)
	dgaGraph.AddVertex(node2)
	dgaGraph.AddVertex(node3)
	dgaGraph.AddVertex(node4)
	dgaGraph.AddEdge(node1, node2)
	dgaGraph.AddEdge(node1, node4)
	dgaGraph.AddEdge(node2, node3)
	dgaGraph.AddEdge(node4, node3)
	bfs := dgaGraph.BFS()
	println(bfs)
}
