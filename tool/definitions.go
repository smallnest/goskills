package tool

import (
	openai "github.com/sashabaranov/go-openai"
)

// GetBaseTools returns the list of base tools available to all skills.
// Following the "Bash is All You Need" philosophy:
// - bash: Universal tool for commands, scripts, file operations via cat/grep/echo
// - tavily_search: Web search via Tavily API (requires API key)
func GetBaseTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "bash",
				Description: "Run a shell command. Use for: file operations (cat, grep, echo, head, tail), running scripts (python3, node, npx tsx, bash), git operations, package management (npm, pip), system commands (ls, find, curl), and any other shell command.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "The shell command to execute.",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "tavily_search",
				Description: "Performs a web search using the Tavily API for the given query and returns a summary of results.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "The search query.",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}
