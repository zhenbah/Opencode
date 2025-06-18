# OpenCode Development Guide

## Build/Test Commands
- `go build -o opencode` - Build the project  
- `go test ./...` - Run all tests
- `go test ./internal/llm/prompt` - Run specific package tests
- `go test -v ./internal/llm/prompt` - Run single test with verbose output
- `go test -run TestFunctionName` - Run specific test function
- `go vet ./...` - Static analysis
- `go mod tidy` - Clean up dependencies
- `./opencode` - Run the built binary

## Code Style Guidelines
- **Imports**: Group stdlib, 3rd party, then internal packages with blank lines between groups
- **Naming**: Use camelCase for private, PascalCase for public; descriptive names preferred
- **Types**: Define custom types for clarity (e.g., `type AgentName string`, `type MCPType string`)
- **Constants**: Group related constants in const blocks with descriptive comments
- **Error Handling**: Always handle errors explicitly; use `require.NoError(t, err)` in tests
- **Testing**: Use testify/assert and testify/require; include `t.Parallel()` for parallel tests
- **Comments**: Package comments start with "Package name"; use descriptive function comments
- **Structure**: Follow standard Go project layout with `internal/` for private packages
- **JSON Tags**: Always include json tags for structs that marshal/unmarshal
- **Context**: Pass context.Context as first parameter for functions that need it
