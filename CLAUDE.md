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

**Note**: All tests should pass. Run `go build ./...` before testing to ensure no compilation errors.

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
5. **Runtime** (`runtime.go`, `runtime_impl.go`): Pipeline execution runtime with process safety
6. **Edge System** (`edge.go`, `edge_impl.go`): DAG edges with conditional expression support
7. **Template Engine** (`templete.go`, `templete_impl.go`): Template rendering for dynamic configuration
8. **Metadata Store** (`metadata.go`, `metadata_impl.go`): Process-safe metadata management
9. **Configuration** (`config.go`): Pipeline configuration structures and parsing

### Key Architecture Patterns

- **DAG-based Pipeline**: Tasks are nodes in a directed acyclic graph with dependencies
- **Executor Pattern**: Different execution backends (Function, Docker, K8s, SSH, Local)
- **Event-driven**: Pipeline and node lifecycle events for monitoring
- **Concurrent Execution**: Independent tasks run in parallel using goroutines
- **Conditional Edges**: Edges support conditional expressions for dynamic execution paths
- **Template Engine**: Support for template rendering in configuration
- **Metadata Management**: Process-safe metadata storage and retrieval during execution

### Directory Structure

- `/` - Core pipeline interfaces and implementations
- `/executor/` - Execution backend implementations
  - `/kubenetes/` - Fully implemented Kubernetes executor
  - `/docker/` - Placeholder for Docker executor
  - `/ssh/` - Placeholder for SSH executor
  - `/local/` - Placeholder for Local executor
- `/test/` - Test suite
- `/doc/` - Documentation files
  - `config.md` - Configuration detailed documentation
  - `templete.md` - Template engine documentation

### Configuration

Pipeline configuration uses YAML format with the following structure:
```yaml
Version: "1.0"           # Configuration version
Name: my-pipeline        # Pipeline name

Metadate:                # Metadata configuration
  type: in-config        # Metadata store type (in-config, redis, http)
  data:                  # Initial metadata key-value pairs
    key1: value1

AI:                      # AI-related configuration
  intent: "描述Pipeline意图"
  constraints:           # Key constraints
    - "约束1"
  template: "template-id"

Param:                   # Pipeline parameters
  key: value

Executors:               # Global executor definitions
  local:
    type: local
    config: {}
  docker:
    type: docker
    config: {}

Logging:                 # Log pushing configuration
  endpoint: http://log-center/api/v1/logs
  headers: {}
  timeout: 5s

Graph: |                 # DAG definition (Mermaid stateDiagram-v2 format)
  stateDiagram-v2
    [*] --> Node1
    Node1 --> Node2

Status:                  # Node runtime status
  Node1: Finished

Nodes:                   # Node definitions with execution details
  NodeName:
    executor: local      # Reference to global executor
    image: optional-image
    steps:               # Multi-step execution
      - name: step1
        run: command
```

See `config.example.yaml` for a complete example.

## Current Implementation Status

**Implemented**:
- ✅ Core pipeline DAG structure and traversal
- ✅ Basic pipeline execution with concurrent processing
- ✅ Kubernetes executor (fully functional)
- ✅ Node state management and event system
- ✅ Cycle detection and graph validation
- ✅ Conditional edges with expression evaluation
- ✅ Template engine for dynamic configuration
- ✅ Metadata store interface and implementations
- ✅ Pipeline Runtime with process safety
- ✅ Multi-step node execution
- ✅ Graph text visualization
- ✅ Log pushing interface

**Planned but not implemented**:
- 🔄 Function executor
- 🔄 Docker executor
- 🔄 SSH executor
- 🔄 Local executor

## Development Notes

- This is a library, not an executable application (no main.go)
- Uses Go 1.23.0 with heavy Kubernetes integration
- The project follows clean architecture with good separation of concerns
- Main development branch is `master`

## Working with the Codebase

When making changes:
1. Understand the DAG traversal algorithm in `pipeline_impl.go`
2. Check the executor interfaces in `executor.go` before implementing new backends
3. Follow the event-driven pattern for pipeline monitoring
4. Ensure graph validation and cycle detection are maintained
5. Run `go build ./...` frequently to catch compilation issues early
6. For conditional logic, check `edge.go` and `eval_context.go`
7. For runtime features, check `runtime.go` and `runtime_impl.go`
8. For template functionality, check `templete.go` and related tests

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **pipelinex** (573 symbols, 1606 relationships, 48 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/pipelinex/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/pipelinex/context` | Codebase overview, check index freshness |
| `gitnexus://repo/pipelinex/clusters` | All functional areas |
| `gitnexus://repo/pipelinex/processes` | All execution flows |
| `gitnexus://repo/pipelinex/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## CLI

- Re-index: `npx gitnexus analyze`
- Check freshness: `npx gitnexus status`
- Generate docs: `npx gitnexus wiki`

<!-- gitnexus:end -->
