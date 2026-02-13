# CLAUDE.md

Please answer in Chinese.

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based CI/CD pipeline execution library that supports multiple execution backends (Docker, Kubernetes, SSH, Local). The library uses a DAG (Directed Acyclic Graph) structure to manage pipeline dependencies and supports concurrent execution of independent tasks.

## Development Commands

### Building
```bash
go build ./...          # Build all packages in the project
go mod tidy             # Clean up dependencies
```

### Testing
```bash
go test ./test/         # Run all tests
go test ./test/ -v      # Run tests with verbose output
```

**Note**: There's currently a compilation error in the test file (`test/pipeline_test.go:76`) where a variable `visited` is declared but not used. This needs to be fixed before running tests.

### Code Quality
```bash
go fmt ./...            # Format all Go code
go vet ./...            # Run static analysis
```

## Architecture Overview

### Core Components

1. **Pipeline Interface** (`pipeline.go`): Main pipeline lifecycle management
2. **DAG Graph Implementation** (`pipeline_impl.go`): Manages task dependencies and traversal
3. **Node System** (`node.go`, `node_impl.go`): Individual task management with state tracking
4. **Executor Pattern** (`executor.go`): Pluggable backend execution system

### Key Architecture Patterns

- **DAG-based Pipeline**: Tasks are nodes in a directed acyclic graph with dependencies
- **Executor Pattern**: Different execution backends (Function, Docker, K8s, SSH, Local)
- **Event-driven**: Pipeline and node lifecycle events for monitoring
- **Concurrent Execution**: Independent tasks run in parallel using goroutines

### Directory Structure

- `/` - Core pipeline interfaces and implementations
- `/executor/` - Execution backend implementations
  - `/kubenetes/` - Fully implemented Kubernetes executor
  - `/docker/` - Placeholder for Docker executor
- `/test/` - Test suite

### Configuration

Pipeline configuration uses YAML format with the following structure:
```yaml
Param:     # Pipeline parameters
Graph: |   # DAG definition (e.g., "Merge->Build\nBuild->Deploy")
Nodes:     # Node definitions with execution details
  NodeName:
    Image: container-image
    Config: { key: value }
    Cmd: command-to-run
```

## Current Implementation Status

**Implemented**:
- ✅ Core pipeline DAG structure and traversal
- ✅ Basic pipeline execution with concurrent processing
- ✅ Kubernetes executor (fully functional)
- ✅ Node state management and event system
- ✅ Cycle detection and graph validation

**Planned but not implemented**:
- 🔄 Function executor
- 🔄 Docker executor
- 🔄 SSH executor
- 🔄 Local executor
- 🔄 Status store interface

## Development Notes

- This is a library, not an executable application (no main.go)
- Uses Go 1.23.0 with heavy Kubernetes integration
- Test compilation currently fails due to unused variable in `test/pipeline_test.go:76`
- The project follows clean architecture with good separation of concerns
- Current development is on the `f-20250430-pipeline-run` feature branch

## Working with the Codebase

When making changes:
1. Understand the DAG traversal algorithm in `pipeline_impl.go`
2. Check the executor interfaces in `executor.go` before implementing new backends
3. Follow the event-driven pattern for pipeline monitoring
4. Ensure graph validation and cycle detection are maintained
5. Run `go build ./...` frequently to catch compilation issues early
