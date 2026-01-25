# Go Skills Tool

A collection of Go tools for AI agent operations following the "Bash is All You Need" philosophy.

## Features

- **Bash Tool**: Universal command execution for file operations, scripts, and system commands
- **Tavily Search**: Web search integration via Tavily API
- **OpenAI Tool Definitions**: Pre-defined tool schemas for AI integration

## Installation

```bash
go get github.com/smallnest/goskills/tool
```

## Usage

### Bash Tool

The `bash` tool is the universal tool for executing shell commands. It can handle:

- **File Operations**: cat, grep, echo, head, tail, find, etc.
- **Script Execution**: python3, node, npx tsx, bash, etc.
- **Git Operations**: git status, git log, git diff, etc.
- **Package Management**: npm, pip, cargo, etc.
- **System Commands**: ls, ps, curl, etc.

```go
import "github.com/smallnest/goskills/tool"

// Execute any shell command
output, err := tool.Bash("ls -la")
if err != nil {
    log.Fatal(err)
}
fmt.Println(output)

// Read a file using bash
content, err := tool.Bash("cat /path/to/file.txt")
if err != nil {
    log.Fatal(err)
}

// Write a file using bash
err = tool.Bash("echo 'Hello, World!' > /path/to/output.txt")
if err != nil {
    log.Fatal(err)
}

// Run a Python script
output, err = tool.Bash("python3 /path/to/script.py")
if err != nil {
    log.Fatal(err)
}

// Run a TypeScript file
output, err = tool.Bash("npx tsx /path/to/file.ts")
if err != nil {
    log.Fatal(err)
}

// Execute with custom timeout
output, err = tool.BashWithTimeout("sleep 30", 10*time.Second)
```

### Tavily Search

```go
// Web search using Tavily API (requires TAVILY_API_KEY environment variable)
result, err := tool.TavilySearch("latest Go programming news")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)

// Search with custom result limit
result, err = tool.TavilySearchWithLimit("Go tutorials", 10)
```

### OpenAI Tool Definitions

```go
// Get base tools for OpenAI integration
tools := tool.GetBaseTools()
for _, t := range tools {
    fmt.Printf("Tool: %s\n", t.Function.Name)
    fmt.Printf("Description: %s\n", t.Function.Description)
}
```

The base tools include:
- `bash`: Universal shell command execution
- `tavily_search`: Web search via Tavily API

## Testing

### Running Tests

```bash
# Run all tests
go test ./tool/... -v

# Run tests with coverage
go test ./tool/... -cover

# Run tests with race detection
go test -race ./tool/...
```

### Test Coverage

The project includes comprehensive unit tests for all tools:
- Bash command execution with safety checks
- Tavily search API integration
- Tool definition validation
- Error handling and timeout behavior

## Environment Variables

- `TAVILY_API_KEY`: Required for Tavily search functionality
- `WORKDIR`: Optional working directory for bash commands (defaults to current directory)

## Safety Features

The bash tool includes built-in safety checks:

1. **Dangerous Command Blocking**: Prevents execution of harmful commands like `rm -rf /`
2. **Timeout Protection**: Default 2-minute timeout to prevent hanging
3. **Working Directory Isolation**: Commands run in a controlled directory

## Dependencies

- `github.com/sashabaranov/go-openai`: OpenAI API integration

## Development

### Prerequisites

- Go 1.25 or later

### Project Structure

```
tool/
├── bash.go              # Bash command execution
├── bash_test.go         # Bash tests
├── search.go            # Tavily search
├── search_test.go       # Search tests
├── definitions.go       # Tool definitions
└── README.md            # This file
```

## Design Philosophy

This project follows the "Bash is All You Need" philosophy inspired by [learn-claude-code](https://github.com/shareAI-lab/learn-claude-code):

> **One tool is enough** - The `bash` tool can execute any shell command, making it unnecessary to have separate tools for file operations, script execution, or system commands. This approach:
>
> - Reduces code complexity
> - Increases flexibility
> - Simplifies maintenance
> - Leverages the power of existing Unix tools

When you need to read a file, use `bash cat file.txt`. When you need to write, use `bash echo 'content' > file.txt`. When you need to search, use `bash grep 'pattern' file`. Simple and universal.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License.
