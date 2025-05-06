package pipelinex

import "errors"

var (
	ErrInvalidGraph = errors.New("invalid graph")
	ErrHasCycle = errors.New("has cycle")
)
