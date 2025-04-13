# Claude Notes

This file contains notes and instructions for Claude when working on this project.

## Commands to Run

- Linting: `go vet ./...`
- Type checking: This is part of the Go compilation process
- Testing: `go test ./...`
- Building: `go build -o phalcon-mcp`

## Project Structure

- `main.go`: Entry point of the application
- `cmd/`: Contains CLI commands
  - `root.go`: Root command definition