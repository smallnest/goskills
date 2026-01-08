package tool

import (
	openai "github.com/sashabaranov/go-openai"
)

// GetBaseTools returns the list of base tools available to all skills.
func GetBaseTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_shell_script",
				Description: "Executes a shell script and returns its combined stdout and stderr. Use this for general shell commands.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"scriptPath": map[string]any{
							"type":        "string",
							"description": "The path to the shell script to execute.",
						},
						"args": map[string]any{
							"type":        "array",
							"description": "A list of string arguments to pass to the script.",
							"items": map[string]any{
								"type": "string",
							},
						},
					},
					"required": []string{"scriptPath"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_python_script",
				Description: "Executes a Python script and returns its combined stdout and stderr.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"scriptPath": map[string]any{
							"type":        "string",
							"description": "The path to the Python script to execute.",
						},
						"args": map[string]any{
							"type":        "array",
							"description": "A list of string arguments to pass to the script.",
							"items": map[string]any{
								"type": "string",
							},
						},
					},
					"required": []string{"scriptPath"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "read_file",
				Description: "Reads the content of a file and returns it as a string.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"filePath": map[string]any{
							"type":        "string",
							"description": "The path to the file to read.",
						},
					},
					"required": []string{"filePath"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "write_file",
				Description: "Writes the given content to a file. If the file does not exist, it will be created. If it exists, its content will be truncated.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"filePath": map[string]any{
							"type":        "string",
							"description": "The path to the file to write.",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The content to write to the file.",
						},
					},
					"required": []string{"filePath", "content"},
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
