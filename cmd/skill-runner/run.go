package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/goskills"
	"github.com/smallnest/goskills/tool" // Import the new tool package
	"github.com/spf13/cobra"
)

var (
	modelName string
	apiBase   string
)

var runCmd = &cobra.Command{
	Use:   "run [prompt]",
	Short: "Processes a user request by selecting and executing a skill.",
	Long: `Processes a user request by simulating the Claude skill-use workflow with an OpenAI-compatible model.
	
This command first discovers all available skills, then asks the LLM to select the most appropriate one.
Finally, it executes the selected skill by feeding its content to the LLM as a system prompt.
If the LLM decides to call a tool, the tool will be executed and its output fed back to the LLM.

Requires the OPENAI_API_KEY environment variable to be set.
You can specify a custom model and API base URL using flags.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userPrompt := strings.Join(args, " ")
		skillsPath := "./examples/skills" // Hardcoded for now

		// --- PRE-FLIGHT CHECK ---
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return errors.New("OPENAI_API_KEY environment variable is not set")
		}

		config := openai.DefaultConfig(apiKey)
		if apiBase != "" {
			config.BaseURL = apiBase
		}
		client := openai.NewClientWithConfig(config)
		ctx := context.Background()

		// --- STEP 1: SKILL DISCOVERY ---
		fmt.Println("🔎 Discovering available skills...")
		availableSkills, err := discoverSkills(skillsPath)
		if err != nil {
			return fmt.Errorf("failed to discover skills: %w", err)
		}
		if len(availableSkills) == 0 {
			return errors.New("no valid skills found")
		}
		fmt.Printf("✅ Found %d skills.\n\n", len(availableSkills))

		// --- STEP 2: SKILL SELECTION ---
		fmt.Println("🧠 Asking LLM to select the best skill...")
		selectedSkillName, err := selectSkill(ctx, client, userPrompt, availableSkills)
		if err != nil {
			return fmt.Errorf("failed during skill selection: %w", err)
		}

		selectedSkill, ok := availableSkills[selectedSkillName]
		if !ok {
			fmt.Printf("⚠️ LLM selected a non-existent skill '%s'. Aborting.\n", selectedSkillName)
			return nil
		}
		fmt.Printf("✅ LLM selected skill: %s\n\n", selectedSkillName)

		// --- STEP 3: SKILL EXECUTION (with Tool Calling) ---
		fmt.Println("🚀 Executing skill (with potential tool calls)...")
		fmt.Println(strings.Repeat("-", 40))

		err = executeSkillWithTools(ctx, client, userPrompt, selectedSkill)
		if err != nil {
			return fmt.Errorf("failed during skill execution: %w", err)
		}

		return nil
	},
}

func discoverSkills(skillsRoot string) (map[string]goskills.SkillPackage, error) {
	skills := make(map[string]goskills.SkillPackage)
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsRoot, entry.Name())
		skillPackage, err := goskills.ParseSkillPackage(skillPath)
		if err != nil {
			continue // Not a valid skill
		}
		skills[skillPackage.Meta.Name] = *skillPackage
	}
	return skills, nil
}

func selectSkill(ctx context.Context, client *openai.Client, userPrompt string, skills map[string]goskills.SkillPackage) (string, error) {
	var sb strings.Builder
	sb.WriteString("User Request: " + "" + userPrompt + "" + "\n\n")
	sb.WriteString("Available Skills:\n")
	for name, skill := range skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", name, skill.Meta.Description))
	}
	sb.WriteString("\nBased on the user request, which single skill is the most appropriate to use? Respond with only the name of the skill.")

	req := openai.ChatCompletionRequest{
		Model: modelName, // Use configurable model name
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are an expert assistant that selects the most appropriate skill to handle a user's request. Your response must be only the exact name of the chosen skill, with no other text or explanation.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: sb.String(),
			},
		},
		Temperature: 0,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	// Clean up the response to get only the skill name
	skillName := strings.TrimSpace(resp.Choices[0].Message.Content)
	skillName = strings.Trim(skillName, `"'`) // Trim quotes and backticks

	return skillName, nil
}

// getToolDefinitions returns the OpenAI tool schemas for the available Go tools.
func getToolDefinitions() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_shell_script",
				Description: "Executes a shell script and returns its combined stdout and stderr. Use this for general shell commands.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"scriptPath": map[string]interface{}{
							"type":        "string",
							"description": "The path to the shell script to execute.",
						},
						"args": map[string]interface{}{
							"type":        "array",
							"description": "A list of string arguments to pass to the script.",
							"items": map[string]interface{}{
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
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"scriptPath": map[string]interface{}{
							"type":        "string",
							"description": "The path to the Python script to execute.",
						},
						"args": map[string]interface{}{
							"type":        "array",
							"description": "A list of string arguments to pass to the script.",
							"items": map[string]interface{}{
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
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"filePath": map[string]interface{}{
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
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"filePath": map[string]interface{}{
							"type":        "string",
							"description": "The path to the file to write.",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The content to write to the file.",
						},
					},
					"required": []string{"filePath", "content"},
				},
			},
		},
		// New tools from langchaingo
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "duckduckgo_search",
				Description: "Performs a DuckDuckGo search for the given query and returns the top results.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query.",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "serpapi_search",
				Description: "Performs a search using SerpAPI for the given query and returns structured results. Requires SERPAPI_API_KEY.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query.",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "metaphor_search",
				Description: "Performs a search for content using Metaphor. Requires METAPHOR_API_KEY.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query.",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "scrape_url",
				Description: "Scrapes the content of a given URL and returns its text.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "The URL to scrape.",
						},
					},
					"required": []string{"url"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "wikipedia_search",
				Description: "Performs a search on Wikipedia for the given query and returns a summary.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query for Wikipedia.",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}

// executeToolCall executes a single tool call and returns its output.
func executeToolCall(toolCall openai.ToolCall) (string, error) {
	var toolOutput string
	var err error

	switch toolCall.Function.Name {
	case "run_shell_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_script arguments: %w", err)
		}
		toolOutput, err = tool.RunShellScript(params.ScriptPath, params.Args)
	case "run_python_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_script arguments: %w", err)
		}
		toolOutput, err = tool.RunPythonScript(params.ScriptPath, params.Args)
	case "read_file":
		var params struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal read_file arguments: %w", err)
		}
		toolOutput, err = tool.ReadFile(params.FilePath)
	case "write_file":
		var params struct {
			FilePath string `json:"filePath"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal write_file arguments: %w", err)
		}
		err = tool.WriteFile(params.FilePath, params.Content)
		if err == nil {
			toolOutput = fmt.Sprintf("Successfully wrote to file: %s", params.FilePath)
		}
	case "duckduckgo_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal duckduckgo_search arguments: %w", err)
		}
		toolOutput, err = tool.DuckDuckGoSearch(params.Query)
	case "wikipedia_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal wikipedia_search arguments: %w", err)
		}
		toolOutput, err = tool.WikipediaSearch(params.Query)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}

	if err != nil {
		return "", fmt.Errorf("tool execution failed for %s: %w", toolCall.Function.Name, err)
	}
	return toolOutput, nil
}

// executeSkillWithTools executes a skill, handling potential tool calls in a loop.
func executeSkillWithTools(ctx context.Context, client *openai.Client, userPrompt string, skill goskills.SkillPackage) error {
	// Reconstruct the skill body from structured parts for the system prompt
	var skillBody strings.Builder
	for _, part := range skill.Body {
		switch p := part.(type) {
		case goskills.TitlePart:
			skillBody.WriteString(fmt.Sprintf("\n# %s\n", p.Text))
		case goskills.SectionPart:
			skillBody.WriteString(fmt.Sprintf("\n## %s\n%s\n", p.Title, p.Content))
		case goskills.MarkdownPart:
			skillBody.WriteString(p.Content + "\n")
		case goskills.ImplementationPart:
			skillBody.WriteString(fmt.Sprintf("\nImplementation in %s:\n", p.Language))
			skillBody.WriteString("```" + p.Language + "\n")
			skillBody.WriteString(p.Code)
			skillBody.WriteString("```\n")
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: skillBody.String(),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}

	availableTools := getToolDefinitions()

	for {
		req := openai.ChatCompletionRequest{
			Model:    modelName, // Use configurable model name
			Messages: messages,
			Tools:    availableTools,
			Stream:   true, // Stream only the final text response
		}

		stream, err := client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			return fmt.Errorf("ChatCompletionStream error: %w", err)
		}
		defer stream.Close()

		var fullResponseContent strings.Builder
		var toolCalls []openai.ToolCall

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break // End of stream
			}
			if err != nil {
				return fmt.Errorf("stream error: %w", err)
			}

			// Accumulate content for final text response
			if response.Choices[0].Delta.Content != "" {
				fullResponseContent.WriteString(response.Choices[0].Delta.Content)
			}

			// Accumulate tool calls
			if response.Choices[0].Delta.ToolCalls != nil {
				for _, tc := range response.Choices[0].Delta.ToolCalls {
					if len(toolCalls) <= *tc.Index {
						toolCalls = append(toolCalls, openai.ToolCall{})
					}
					if tc.ID != "" {
						toolCalls[*tc.Index].ID = tc.ID
					}
					if tc.Type != "" {
						toolCalls[*tc.Index].Type = tc.Type
					}
					if tc.Function.Name != "" {
						toolCalls[*tc.Index].Function.Name = tc.Function.Name
					}
					toolCalls[*tc.Index].Function.Arguments += tc.Function.Arguments
				}
			}
		}

		// If there's a text response, print it and we're done
		if fullResponseContent.Len() > 0 {
			fmt.Print(fullResponseContent.String())
			fmt.Println()
			return nil
		}

		// If there are tool calls, execute them
		if len(toolCalls) > 0 {
			fmt.Println("\n--- LLM requested tool calls ---")
			for _, tc := range toolCalls {
				fmt.Printf("⚙️ Calling tool: %s with args: %s\n", tc.Function.Name, tc.Function.Arguments)
				toolOutput, err := executeToolCall(tc)
				if err != nil {
					fmt.Printf("❌ Tool call failed: %v\n", err)
					// Add error message to history and let LLM try to recover
					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						ToolCallID: tc.ID,
						Content:    fmt.Sprintf("Error: %v", err),
					})
				} else {
					fmt.Printf("✅ Tool output: %s\n", toolOutput)
					// Add tool call and output to history
					messages = append(messages, openai.ChatCompletionMessage{
						Role:      openai.ChatMessageRoleAssistant,
						ToolCalls: []openai.ToolCall{tc}, // Add the tool call made by the assistant
					})
					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						ToolCallID: tc.ID,
						Content:    toolOutput,
					})
				}
			}
			fmt.Println("--- Continuing LLM conversation ---")
			// Loop again to let LLM process tool output
		} else {
			// Should not happen if fullResponseContent is empty and no tool calls
			return errors.New("LLM response was empty and contained no tool calls")
		}
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&modelName, "model", "m", "gpt-4o", "OpenAI-compatible model name to use")
	runCmd.Flags().StringVarP(&apiBase, "api-base", "b", "", "OpenAI-compatible API base URL (e.g., https://api.openai.com/v1)")
}
