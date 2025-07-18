# OpenCode.md

## Build, Lint, and Test Commands

- **Build project:**  
  `go build -o opencode ./main.go`
- **Run (local dev binary):**  
  `./opencode`
- **Lint project:**  
  `go fmt ./... && go vet ./...`
- **Run all tests:**  
  `go test ./...`
- **Run a single test (Go style):**  
  `go test -run TestFunctionName ./path/to/package`

## Code Style Guidelines

- **Imports:**  
  - Organized in standard, external, and local blocks, separated by blank lines.  
  - Group related imports, use `goimports` or `go fmt` to maintain order.
- **Formatting:**  
  - Use `go fmt` for code formatting before commits.  
  - Prefer tabs over spaces for indentation.  
  - 120-character line limit is a guideline, not enforced.
- **Types & Naming:**  
  - Use CamelCase for types, structs, and exported functions (e.g., `MyType`, `NewSession`).  
  - Unexported identifiers are lowerCamelCase (e.g., `doThing`).  
  - Acronyms should be uppercase (e.g., `DB`, `ID`).  
  - Constants in ALL_CAPS_SNAKE_CASE only if used as enums.
- **Error Handling:**  
  - Always check for returned `error`s; do not ignore errors.  
  - Use `fmt.Errorf` for error wrapping with context, or `%w` for error chaining.  
  - Use sentinel error variables where needed (e.g., `var ErrNotFound = errors.New("not found")`).
- **Folder Structure:**  
  - `cmd/`: Command-line entry point definitions.  
  - `internal/`: Most business and app logic, organized into functional packages (e.g., `db`, `llm`, `config`, `tui`).  
  - Tests are placed alongside the code as `_test.go` files.  
  - `scripts/`: Helper scripts for release and maintenance.
- **Tests:**  
  - Test functions start with `Test` and are exported (capitalized) in `_test.go` files.  
  - Use Go's standard `testing` package.
- **Other Practices:**  
  - Use context where practical for timeouts/cancellation.  
  - Prefer composition over inheritance.  
  - Avoid global mutable state; use dependency injection or explicit context.
- **Documentation:**  
  - Exported functions and types have doc comments.  
  - README.md gives a project overview; add context-specific docs as needed.

## Cursor and Copilot Rules

No explicit Cursor or Copilot configuration or rules detected. Future rules could be added in `.cursor.toml`, `.copilot/`, or similar config files.
