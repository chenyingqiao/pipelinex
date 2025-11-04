package test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/chenyingqiao/pipelinex"
)

// TestNewRuntime tests creating a new Runtime instance
func TestNewRuntime(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	if runtime == nil {
		t.Fatal("NewRuntime should not return nil")
	}

	// Check if it's a RuntimeImpl type using reflection
	if reflect.TypeOf(runtime).String() != "*pipelinex.RuntimeImpl" {
		t.Fatalf("NewRuntime should return *RuntimeImpl, got %v", reflect.TypeOf(runtime))
	}
}

// TestRuntimeImpl_Get tests getting a pipeline
func TestRuntimeImpl_Get(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Test getting non-existent pipeline
	_, err := runtime.Get("non-existent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent pipeline")
	}
}

// TestRuntimeImpl_RunSync tests synchronous pipeline execution
func TestRuntimeImpl_RunSync(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Prepare test configuration
	config := `
Param:
  test-param: "test-value"
Graph: "Task1->Task2"
Nodes:
  Task1:
    Image: "test-image:latest"
    Config:
      key1: "value1"
    Cmd: "echo 'task1'"
  Task2:
    Image: "test-image:latest"
    Config:
      key2: "value2"
    Cmd: "echo 'task2'"
`

	// Create test listener
	listener := &TestListener{}

	// Execute synchronous pipeline
	pipeline, err := runtime.RunSync(ctx, "test-sync-pipeline", config, listener)
	if err != nil {
		t.Fatalf("RunSync failed: %v", err)
	}

	if pipeline == nil {
		t.Fatal("Pipeline should not be nil")
	}

	// Check if pipeline is cleaned up
	_, err = runtime.Get("test-sync-pipeline")
	if err == nil {
		t.Fatal("Pipeline should be cleaned up after sync execution")
	}
}

// TestRuntimeImpl_RunSync_InvalidConfig tests synchronous execution with invalid config
func TestRuntimeImpl_RunSync_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Prepare invalid configuration
	invalidConfig := `
invalid: yaml: content
  test-param: "test-value"
  missing: closing: brace
`

	_, err := runtime.RunSync(ctx, "test-invalid-config", invalidConfig, nil)
	if err == nil {
		t.Fatal("Expected error with invalid YAML config")
	}
}

// TestRuntimeImpl_RunSync_DuplicateID tests synchronous execution with duplicate ID
func TestRuntimeImpl_RunSync_DuplicateID(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Prepare test configuration
	config := `
Param:
  test-param: "test-value"
Nodes:
  Task1:
    Image: "test-image:latest"
    Cmd: "echo 'task1'"
`

	// First execution
	_, err := runtime.RunSync(ctx, "duplicate-id", config, nil)
	if err != nil {
		t.Fatalf("First RunSync failed: %v", err)
	}

	// Second execution with same ID
	_, err = runtime.RunSync(ctx, "duplicate-id", config, nil)
	if err == nil {
		t.Fatal("Expected error when running pipeline with duplicate ID")
	}
}

// TestRuntimeImpl_RunAsync tests asynchronous pipeline execution
func TestRuntimeImpl_RunAsync(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Prepare test configuration
	config := `
Param:
  test-param: "test-value"
Nodes:
  Task1:
    Image: "test-image:latest"
    Cmd: "echo 'task1'"
`

	// Create test listener
	listener := &TestListener{}

	// Execute asynchronous pipeline
	pipeline, err := runtime.RunAsync(ctx, "test-async-pipeline", config, listener)
	if err != nil {
		t.Fatalf("RunAsync failed: %v", err)
	}

	if pipeline == nil {
		t.Fatal("Pipeline should not be nil")
	}

	// Check if pipeline is stored in runtime
	retrieved, err := runtime.Get("test-async-pipeline")
	if err != nil {
		t.Fatalf("Pipeline should be stored in runtime: %v", err)
	}
	if retrieved != pipeline {
		t.Fatal("Retrieved pipeline should be the same instance")
	}

	// Wait for async execution to complete
	select {
	case <-pipeline.Done():
		// Pipeline completed
	case <-time.After(5 * time.Second):
		// Cancel pipeline
		runtime.Cancel(ctx, "test-async-pipeline")
	}
}

// TestRuntimeImpl_Cancel tests pipeline cancellation
func TestRuntimeImpl_Cancel(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Prepare test configuration
	config := `
Param:
  test-param: "test-value"
Nodes:
  Task1:
    Image: "test-image:latest"
    Cmd: "sleep 10"
`

	// Execute asynchronous pipeline
	pipeline, err := runtime.RunAsync(ctx, "test-cancel-pipeline", config, nil)
	if err != nil {
		t.Fatalf("RunAsync failed: %v", err)
	}

	// Cancel pipeline
	err = runtime.Cancel(ctx, "test-cancel-pipeline")
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	// Wait for pipeline to be cancelled
	select {
	case <-pipeline.Done():
		// Pipeline was cancelled successfully
	case <-time.After(2 * time.Second):
		t.Fatal("Pipeline should be cancelled quickly")
	}
}

// TestRuntimeImpl_Cancel_NonExistent tests cancelling non-existent pipeline
func TestRuntimeImpl_Cancel_NonExistent(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	err := runtime.Cancel(ctx, "non-existent-pipeline")
	if err == nil {
		t.Fatal("Expected error when cancelling non-existent pipeline")
	}
}

// TestRuntimeImpl_Rm tests pipeline removal
func TestRuntimeImpl_Rm(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Remove pipeline (this should not panic even if pipeline doesn't exist)
	runtime.Rm("test-rm-pipeline")
}

// TestRuntimeImpl_Done tests Done channel
func TestRuntimeImpl_Done(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	doneChan := runtime.Done()
	if doneChan == nil {
		t.Fatal("Done channel should not be nil")
	}

	// Test if channel is closed after StopBackground
	select {
	case <-doneChan:
		t.Fatal("Done channel should not be closed initially")
	default:
		// Normal case, channel not closed
	}

	runtime.StopBackground()

	// Wait a bit to ensure channel is closed
	select {
	case <-doneChan:
		// Channel closed as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Done channel should be closed after StopBackground")
	}
}

// TestRuntimeImpl_Notify tests notification functionality
func TestRuntimeImpl_Notify(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Test string notification
	err := runtime.Notify("test message")
	if err != nil {
		t.Fatalf("Notify with string failed: %v", err)
	}

	// Test map notification
	err = runtime.Notify(map[string]interface{}{
		"message": "test map message",
		"type":    "info",
	})
	if err != nil {
		t.Fatalf("Notify with map failed: %v", err)
	}

	// Test other type notification
	err = runtime.Notify(123)
	if err != nil {
		t.Fatalf("Notify with number failed: %v", err)
	}
}

// TestRuntimeImpl_Ctx tests context
func TestRuntimeImpl_Ctx(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	retrievedCtx := runtime.Ctx()
	if retrievedCtx == nil {
		t.Fatal("Context should not be nil")
	}

	// Test if context is cancellable
	if retrievedCtx.Done() == nil {
		t.Fatal("Context should be cancellable")
	}
}

// TestRuntimeImpl_StopBackground tests stopping background processing
func TestRuntimeImpl_StopBackground(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Start background processing
	runtime.StartBackground()

	// Wait a bit for background processing to start
	time.Sleep(10 * time.Millisecond)

	// Stop background processing
	runtime.StopBackground()

	// Test if context is cancelled
	select {
	case <-runtime.Ctx().Done():
		// Context cancelled as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled after StopBackground")
	}
}

// TestRuntimeImpl_ConcurrentAccess tests concurrent access
func TestRuntimeImpl_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	runtime := pipelinex.NewRuntime(ctx)

	// Prepare test configuration
	config := `
Param:
  test-param: "test-value"
Nodes:
  Task%d:
    Image: "test-image:latest"
    Cmd: "echo 'task%d'"
`

	// Concurrent test
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			pipelineConfig := fmt.Sprintf(config, id, id)
			_, err := runtime.RunAsync(ctx, fmt.Sprintf("pipeline-%d", id), pipelineConfig, nil)
			if err != nil {
				t.Errorf("Concurrent RunAsync failed: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestListener test listener implementation
type TestListener struct{}

func (l *TestListener) Handle(p pipelinex.Pipeline, event pipelinex.Event) {
	// Simple implementation that does nothing for testing
}

func (l *TestListener) Events() []pipelinex.Event {
	return []pipelinex.Event{
		pipelinex.PipelineInit,
		pipelinex.PipelineStart,
		pipelinex.PipelineFinish,
		pipelinex.PipelineNodeStart,
		pipelinex.PipelineNodeFinish,
	}
}
