package pipelinex

import (
	"context"
	"time"
)

// Level 日志级别
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Entry 单条日志
type Entry struct {
	Pipeline  string    `json:"pipeline"`
	BuildID   string    `json:"buildId"`
	Node      string    `json:"node"`
	Step      string    `json:"step"`
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
	Output    string    `json:"output"` // 命令标准输出/错误
}

// Pusher 日志推送接口
type Pusher interface {
	// Push 推送单条日志
	Push(ctx context.Context, entry Entry) error

	// PushBatch 批量推送
	PushBatch(ctx context.Context, entries []Entry) error

	// Close 关闭连接，刷新缓冲
	Close() error
}
