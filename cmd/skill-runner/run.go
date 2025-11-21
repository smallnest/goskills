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
	"github.com/smallnest/goskills/config" // Import the new config package
	"github.com/smallnest/goskills/tool"   // Import the new tool package
	"github.com/spf13/cobra"
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
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		userPrompt := strings.Join(args, " ")
		// -- COPY STDIN TO SUPPORT FILE --
		if len(args) == 0 {
			userPromptBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			userPrompt = strings.TrimSpace(string(userPromptBytes))
		}

		// --- LOAD CONFIG ---
		cfg, err := config.LoadConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// --- PRE-FLIGHT CHECK ---
		if cfg.APIKey == "" {
			return errors.New("OPENAI_API_KEY environment variable is not set")
		}

		openaiConfig := openai.DefaultConfig(cfg.APIKey)
		if cfg.APIBase != "" {
			openaiConfig.BaseURL = cfg.APIBase
		}
		client := openai.NewClientWithConfig(openaiConfig)
		ctx := context.Background()

		// --- STEP 1: SKILL DISCOVERY ---
		if cfg.Verbose {
			fmt.Printf("🔎 Discovering available skills in %s...\n", cfg.SkillsDir)
		}
		availableSkills, err := discoverSkills(cfg.SkillsDir)
		if err != nil {
			return fmt.Errorf("failed to discover skills: %w", err)
		}
		if len(availableSkills) == 0 {
			return errors.New("no valid skills found")
		}
		fmt.Printf("✅ Found %d skills.\n\n", len(availableSkills))

		// --- STEP 2: SKILL SELECTION ---
		fmt.Println("🧠 Asking LLM to select the best skill...")
		selectedSkillName, err := selectSkill(ctx, client, cfg, userPrompt, availableSkills)
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

		err = executeSkillWithTools(ctx, client, cfg, userPrompt, selectedSkill)
		if err != nil {
			return fmt.Errorf("failed during skill execution: %w", err)
		}

		return nil
	},
}

func discoverSkills(skillsRoot string) (map[string]goskills.SkillPackage, error) {
	packages, err := goskills.ParseSkillPackages(skillsRoot)
	if err != nil {
		return nil, err
	}

	skills := make(map[string]goskills.SkillPackage, len(packages))
	for _, pkg := range packages {
		if pkg != nil {
			skills[pkg.Meta.Name] = *pkg
		}
	}

	return skills, nil
}

func selectSkill(ctx context.Context, client *openai.Client, cfg *config.Config, userPrompt string, skills map[string]goskills.SkillPackage) (string, error) {
	var sb strings.Builder
	sb.WriteString("User Request: " + "" + userPrompt + "" + "\n\n")
	sb.WriteString("Available Skills:\n")
	for name, skill := range skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", name, skill.Meta.Description))
	}
	sb.WriteString("\nBased on the user request, which single skill is the most appropriate to use? Respond with only the name of the skill.")

	req := openai.ChatCompletionRequest{
		Model: cfg.Model, // Use configurable model name
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

// executeToolCall executes a single tool call and returns its output.
func executeToolCall(toolCall openai.ToolCall, scriptMap map[string]string, skillPath string) (string, error) {
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

		// Resolve path relative to skill directory if it's not absolute and skillPath is provided
		path := params.FilePath
		if !filepath.IsAbs(path) && skillPath != "" {
			resolvedPath := filepath.Join(skillPath, path)
			// Check if file exists at resolved path
			if _, err := os.Stat(resolvedPath); err == nil {
				path = resolvedPath
			}
		}

		toolOutput, err = tool.ReadFile(path)
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
		// Check if it's a generated script tool
		if scriptPath, ok := scriptMap[toolCall.Function.Name]; ok {
			var params struct {
				Args []string `json:"args"`
			}
			// Arguments might be optional or empty
			if toolCall.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
					return "", fmt.Errorf("failed to unmarshal script arguments: %w", err)
				}
			}

			// Determine if python or shell based on extension
			if strings.HasSuffix(scriptPath, ".py") {
				toolOutput, err = tool.RunPythonScript(scriptPath, params.Args)
			} else {
				toolOutput, err = tool.RunShellScript(scriptPath, params.Args)
			}
		} else {
			return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
		}
	}

	if err != nil {
		return "", fmt.Errorf("tool execution failed for %s: %w", toolCall.Function.Name, err)
	}
	return toolOutput, nil
}

// executeSkillWithTools executes a skill, handling potential tool calls in a loop.
func executeSkillWithTools(ctx context.Context, client *openai.Client, cfg *config.Config, userPrompt string, skill goskills.SkillPackage) error {
	// Reconstruct the skill body from structured parts for the system prompt
	var skillBody strings.Builder
	skillBody.WriteString(skill.Body) // Directly use the raw markdown body
	skillBody.WriteString("\n\n")

	// --- INJECT SKILL CONTEXT ---
	skillBody.WriteString("## SKILL CONTEXT\n")
	skillBody.WriteString(fmt.Sprintf("Skill Root Path: %s\n", skill.Path))
	skillBody.WriteString("Available Resources:\n")
	if len(skill.Resources.Scripts) > 0 {
		skillBody.WriteString("- Scripts:\n")
		for _, s := range skill.Resources.Scripts {
			skillBody.WriteString(fmt.Sprintf("  - %s\n", s))
		}
	}
	if len(skill.Resources.Templates) > 0 {
		skillBody.WriteString("- Templates:\n")
		for _, t := range skill.Resources.Templates {
			skillBody.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if len(skill.Resources.References) > 0 {
		skillBody.WriteString("- References:\n")
		for _, r := range skill.Resources.References {
			skillBody.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}
	if len(skill.Resources.Assets) > 0 {
		skillBody.WriteString("- Assets:\n")
		for _, a := range skill.Resources.Assets {
			skillBody.WriteString(fmt.Sprintf("  - %s\n", a))
		}
	}
	skillBody.WriteString("\nIMPORTANT: When reading resource files mentioned in the skill definition, you must use the full path or a path relative to the Skill Root Path.\n")

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

	availableTools, scriptMap := goskills.GenerateToolDefinitions(skill)

	// --- DEBUG: Print Available Tools ---
	fmt.Println("🛠️  Available Tools:")
	for _, t := range availableTools {
		fmt.Printf("  - %s\n", t.Function.Name)
	}
	fmt.Println(strings.Repeat("-", 40))

	for {
		req := openai.ChatCompletionRequest{
			Model:    cfg.Model, // Use configurable model name
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

		// Print text response if any
		if fullResponseContent.Len() > 0 {
			fmt.Print(fullResponseContent.String())
			fmt.Println()
		}

		// If there are tool calls, execute them
		if len(toolCalls) > 0 {
			fmt.Println("\n--- LLM requested tool calls ---")

			// Append the assistant's message (with text and tool calls) to history
			assistantMsg := openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   fullResponseContent.String(),
				ToolCalls: toolCalls,
			}
			messages = append(messages, assistantMsg)

			for _, tc := range toolCalls {
				fmt.Printf("⚙️ Calling tool: %s with args: %s\n", tc.Function.Name, tc.Function.Arguments)

				// --- SECURITY CHECK ---
				// 1. Allowlist Check
				if len(cfg.AllowedScripts) > 0 {
					allowed := false
					for _, script := range cfg.AllowedScripts {
						if script == tc.Function.Name {
							allowed = true
							break
						}
					}
					if !allowed {
						fmt.Printf("❌ Tool execution denied: '%s' is not in the allowlist.\n", tc.Function.Name)
						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							ToolCallID: tc.ID,
							Content:    fmt.Sprintf("Error: Tool '%s' is not allowed by configuration.", tc.Function.Name),
						})
						continue
					}
				}

				// 2. Confirmation Prompt
				if !cfg.AutoApproveTools {
					fmt.Print("⚠️  Allow this tool execution? [y/N]: ")
					var input string
					fmt.Scanln(&input)
					if strings.ToLower(input) != "y" {
						fmt.Println("❌ Tool execution denied by user.")
						messages = append(messages, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							ToolCallID: tc.ID,
							Content:    "Error: User denied tool execution.",
						})
						continue
					}
				}

				toolOutput, err := executeToolCall(tc, scriptMap, skill.Path)
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
					// Add tool output to history
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
			// If no tool calls and we have text, we are done
			if fullResponseContent.Len() > 0 {
				return nil
			}
			// Should not happen if fullResponseContent is empty and no tool calls
			return errors.New("LLM response was empty and contained no tool calls")
		}
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	config.SetupFlags(runCmd)
}
