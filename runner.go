package goskills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	openai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/goskills/log"
	"github.com/smallnest/goskills/mcp"
	"github.com/smallnest/goskills/tool"
)

// OpenAIChatClient interface for dependency injection and testing
type OpenAIChatClient interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// Agent manages the skill discovery, selection, and execution process.
type Agent struct {
	client    OpenAIChatClient
	cfg       RunnerConfig
	messages  []openai.ChatCompletionMessage // Stores the conversation history
	mcpClient *mcp.Client
}

// RunnerConfig holds all the necessary configuration for the runner.
type RunnerConfig struct {
	APIKey           string
	APIBase          string
	Model            string
	SkillsDir        string
	Verbose          int
	Debug            bool
	AutoApproveTools bool
	AllowedScripts   []string
	Loop             bool
	SkillName        string
}

// NewAgent creates and initializes a new Agent.
func NewAgent(cfg RunnerConfig, mcpClient *mcp.Client) (*Agent, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is not set")
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o" // Default model
	}

	openaiConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.APIBase != "" {
		openaiConfig.BaseURL = cfg.APIBase
	}
	client := openai.NewClientWithConfig(openaiConfig)

	return &Agent{
		client:    client,
		cfg:       cfg,
		messages:  []openai.ChatCompletionMessage{}, // Initialize empty message history
		mcpClient: mcpClient,
	}, nil
}

// Run executes the main skill selection and execution logic for a single turn.
func (a *Agent) Run(ctx context.Context, userPrompt string) (string, error) {
	selectedSkill, err := a.selectAndPrepareSkill(ctx, userPrompt)
	if err != nil {
		return "", err
	}

	// --- STEP 3: SKILL EXECUTION (with Tool Calling) ---
	if a.cfg.Verbose >= 1 {
		log.Info("executing skill (with potential tool calls)")
		log.Info(strings.Repeat("-", 10) + "Selected Skill:" + selectedSkill.Meta.Name + strings.Repeat("-", 10))
	}

	return a.executeSkillWithTools(ctx, userPrompt, selectedSkill)
}

// RunLoop starts an interactive session for a selected skill.
func (a *Agent) RunLoop(ctx context.Context, initialPrompt string) error {
	selectedSkill, err := a.selectAndPrepareSkill(ctx, initialPrompt)
	if err != nil {
		return err
	}

	// Initialize the system message before entering the loop
	var skillBody strings.Builder
	skillBody.WriteString("## SELECTED SKILL\n")
	skillBody.WriteString(fmt.Sprintf("Name: %s\n", selectedSkill.Meta.Name))
	skillBody.WriteString(fmt.Sprintf("Description: %s\n", selectedSkill.Meta.Description))
	skillBody.WriteString(fmt.Sprintf("Version: %s\n\n", selectedSkill.Meta.Version))
	skillBody.WriteString("## SKILL INSTRUCTIONS\n")
	skillBody.WriteString(selectedSkill.Body)
	skillBody.WriteString("\n\n##如果SKILL中没有要调用脚本的必要，则不要调用Tool,尤其是run_shell_script工具，直接根据SKILL的描述直接生成答案。\n\n ## SKILL CONTEXT\n")
	skillBody.WriteString(fmt.Sprintf("Skill Root Path: %s\n", selectedSkill.Path))
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: skillBody.String(),
	})

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "▶ ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	currentPrompt := initialPrompt

	for {
		finalOutput, err := a.continueSkillWithTools(ctx, currentPrompt, selectedSkill)
		if err != nil {
			log.Error("error during execution: %v", err)
		} else {
			log.Info("final output:")
			log.Info("%s", finalOutput)
		}

		fmt.Print("\n")
		currentPrompt, err = rl.Readline()
		if err != nil {
			break
		}
		currentPrompt = strings.TrimSpace(currentPrompt)

		if currentPrompt == "" || strings.EqualFold(currentPrompt, "q") || strings.EqualFold(currentPrompt, "quit") || strings.EqualFold(currentPrompt, "exit") {
			break
		}
	}
	return nil
}

// selectAndPrepareSkill discovers and selects the appropriate skill.
func (a *Agent) selectAndPrepareSkill(ctx context.Context, userPrompt string) (*SkillPackage, error) {
	// --- STEP 1: SKILL DISCOVERY ---
	if a.cfg.Verbose >= 1 {
		log.Info("discovering available skills in %s...", a.cfg.SkillsDir)
	}
	availableSkills, err := a.discoverSkills(a.cfg.SkillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover skills: %w", err)
	}
	if len(availableSkills) == 0 {
		return nil, errors.New("no valid skills found")
	}
	if a.cfg.Verbose >= 1 {
		log.Info("found %d skills", len(availableSkills))
	}

	// --- STEP 2: SKILL SELECTION ---
	var selectedSkillName string

	// If skill is explicitly specified via --skill flag, use it directly
	if a.cfg.SkillName != "" {
		selectedSkillName = a.cfg.SkillName
		if a.cfg.Verbose >= 1 {
			log.Info("using explicitly specified skill: %s", selectedSkillName)
		}
	} else {
		// Otherwise, ask LLM to select the best skill
		if a.cfg.Verbose >= 1 {
			log.Info("asking llm to select the best skill")
		}
		selectedSkillName, err = a.selectSkill(ctx, userPrompt, availableSkills)
		if err != nil {
			return nil, fmt.Errorf("failed during skill selection: %w", err)
		}
		if a.cfg.Verbose >= 1 {
			log.Info("llm selected skill: %s", selectedSkillName)
		}
	}

	selectedSkill, ok := availableSkills[selectedSkillName]
	if !ok {
		return nil, fmt.Errorf("skill '%s' not found. Available skills: %v", selectedSkillName, getAvailableSkillNames(availableSkills))
	}
	if a.cfg.Verbose >= 1 {
		log.Info("selected skill: %s", selectedSkillName)
	}
	return &selectedSkill, nil
}

// getAvailableSkillNames returns a slice of available skill names for error messages
func getAvailableSkillNames(skills map[string]SkillPackage) []string {
	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	return names
}

func (a *Agent) discoverSkills(skillsRoot string) (map[string]SkillPackage, error) {
	packages, err := ParseSkillPackages(skillsRoot)
	if err != nil {
		return nil, err
	}

	skills := make(map[string]SkillPackage, len(packages))
	for _, pkg := range packages {
		if pkg != nil {
			skills[pkg.Meta.Name] = *pkg
		}
	}

	return skills, nil
}

func (a *Agent) selectSkill(ctx context.Context, userPrompt string, skills map[string]SkillPackage) (string, error) {
	var sb strings.Builder
	sb.WriteString("User Request: " + "" + userPrompt + "" + "\n\n")
	sb.WriteString("Available Skills:\n")
	for name, skill := range skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", name, skill.Meta.Description))
	}
	sb.WriteString("\nSelection Guidelines:\n")
	sb.WriteString("- For pure mathematical calculations (arithmetic, trigonometry, logarithms, etc.), ALWAYS prefer 'calculator-skill' over spreadsheet skills\n")
	sb.WriteString("- Only choose spreadsheet skills (xlsx, csv) when the user needs to create/read/modify spreadsheet FILES\n")
	sb.WriteString("- Function names that happen to exist in Excel do NOT make it a spreadsheet task\n")
	sb.WriteString("\nBased on the user request and guidelines above, which single skill is the most appropriate to use?")
	sb.WriteString("\n\nIMPORTANT: You MUST select exactly one skill from the above list, even if the request seems simple. Respond with ONLY the skill name, nothing else. Do not explain your choice or answer the question directly.")

	skillPrompt := SkillsToPrompt(skills)

	// Use a temporary message history for skill selection
	selectionMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a skill selection assistant. Your ONLY job is to select the most appropriate skill from the available list. You must ALWAYS choose exactly one skill - never refuse to select or try to answer the question yourself.\n" + skillPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: sb.String(),
		},
	}

	req := openai.ChatCompletionRequest{
		Model:       a.cfg.Model,
		Messages:    selectionMessages,
		Temperature: 0,
	}

	a.debugPrintRequest(req)
	resp, err := a.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	a.debugPrintResponse(resp)

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.Trim(content, "'\"")

	// Extract just the skill name if there's extra text
	// Look for skill names in the content
	skillName := extractSkillName(content, skills)

	if a.cfg.Verbose >= 1 {
		fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
		fmt.Fprintf(os.Stderr, "Selected Skill: %s\n", skillName)
		fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	}

	return skillName, nil
}

// extractSkillName extracts the skill name from AI response content
func extractSkillName(content string, skills map[string]SkillPackage) string {
	// First, check if the content is already a valid skill name
	if _, exists := skills[content]; exists {
		return content
	}

	// Convert content to lowercase for case-insensitive matching
	lowerContent := strings.ToLower(content)

	// Look for any skill name mentioned in the content
	for skillName := range skills {
		// Check exact match (case-insensitive)
		if strings.Contains(lowerContent, strings.ToLower(skillName)) {
			return skillName
		}
	}

	// If no skill name found, return the original content
	// This preserves the existing behavior when no skills match
	return content
}

// debugPrintRequest prints the LLM request in debug mode
func (a *Agent) debugPrintRequest(req openai.ChatCompletionRequest) {
	if a.cfg.Verbose < 2 {
		return
	}
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	fmt.Fprintln(os.Stderr, "LLM Request:")
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	jsonBytes, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request to JSON: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	}
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
}

// debugPrintResponse prints the LLM response in debug mode
func (a *Agent) debugPrintResponse(resp openai.ChatCompletionResponse) {
	if a.cfg.Verbose < 2 {
		return
	}
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	fmt.Fprintln(os.Stderr, "LLM Response:")
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	jsonBytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response to JSON: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	}
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
}

// executeSkillWithTools sets up the initial system prompt and starts the tool-use conversation.
func (a *Agent) executeSkillWithTools(ctx context.Context, userPrompt string, skill *SkillPackage) (string, error) {
	// Prepare the system message once
	var skillBody strings.Builder
	skillBody.WriteString("## SELECTED SKILL\n")
	skillBody.WriteString(fmt.Sprintf("Name: %s\n", skill.Meta.Name))
	skillBody.WriteString(fmt.Sprintf("Description: %s\n", skill.Meta.Description))
	skillBody.WriteString(fmt.Sprintf("Version: %s\n\n", skill.Meta.Version))
	skillBody.WriteString("## SKILL INSTRUCTIONS\n")
	skillBody.WriteString(skill.Body)
	skillBody.WriteString("\n\n##如果SKILL中没有要调用脚本的必要，则不要调用Tool,尤其是run_shell_script工具，直接根据SKILL的描述直接生成答案。\n\n ## SKILL CONTEXT\n")
	skillBody.WriteString(fmt.Sprintf("Skill Root Path: %s\n", skill.Path))
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: skillBody.String(),
	})

	return a.continueSkillWithTools(ctx, userPrompt, skill)
}

// continueSkillWithTools continues a conversation with a new user prompt.
func (a *Agent) continueSkillWithTools(ctx context.Context, userPrompt string, skill *SkillPackage) (string, error) {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userPrompt,
	})

	availableTools, scriptMap := GenerateToolDefinitions(skill)

	// Add MCP tools if client is available
	if a.mcpClient != nil {
		mcpTools, err := a.mcpClient.GetTools(ctx)
		if err != nil {
			log.Warn("failed to get mcp tools: %v", err)
		} else {
			availableTools = append(availableTools, mcpTools...)
		}
	}

	var finalResponse strings.Builder

	for range 20 { // Limit to 20 iterations to prevent infinite loops
		req := openai.ChatCompletionRequest{
			Model:    a.cfg.Model,
			Messages: a.messages, // Use agent's messages
			Tools:    availableTools,
		}

		a.debugPrintRequest(req)
		resp, err := a.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("ChatCompletion error: %w", err)
		}
		a.debugPrintResponse(resp)

		msg := resp.Choices[0].Message
		a.messages = append(a.messages, msg) // Append LLM's response

		if msg.ToolCalls == nil {
			finalResponse.WriteString(msg.Content)
			return finalResponse.String(), nil
		}

		for _, tc := range msg.ToolCalls {
			if a.cfg.Verbose >= 1 {
				log.Info("calling tool: %s with args: %s", tc.Function.Name, tc.Function.Arguments)
			}

			if !a.cfg.AutoApproveTools {
				fmt.Print("⚠️  Allow this tool execution? [y/N]: ")
				var input string
				if _, err := fmt.Scanln(&input); err != nil {
					// Handle scan error, default to denying
					log.Error("tool execution denied due to input error: %v", err)
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						ToolCallID: tc.ID,
						Content:    "Error: Input scanning failed.",
					})
					continue
				}
				if strings.ToLower(strings.TrimSpace(input)) != "y" {
					log.Info("tool execution denied by user: %s", tc.Function.Name)
					a.messages = append(a.messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						ToolCallID: tc.ID,
						Content:    "Error: User denied tool execution.",
					})
					continue
				}
			}

			var toolOutput string
			var err error

			// Check if it is an MCP tool
			if a.mcpClient != nil && strings.Contains(tc.Function.Name, "__") {
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					toolOutput = fmt.Sprintf("Error unmarshalling arguments: %v", err)
				} else {
					var result any
					result, err = a.mcpClient.CallTool(ctx, tc.Function.Name, args)
					if err == nil {
						// Convert result to string/JSON
						resBytes, _ := json.Marshal(result)
						toolOutput = string(resBytes)
					}
				}
			} else {
				toolOutput, err = a.executeToolCall(tc, scriptMap, skill.Path)
			}

			if err != nil {
				log.Error("tool call failed: %v", err)
				// Provide detailed error information to help LLM understand what went wrong
				errorMsg := fmt.Sprintf("Tool execution failed: %s\nError details: %v\nTool name: %s\nArguments: %s\n\nYou can try:\n1. Retry with different parameters\n2. Use a different tool to fix it\n3. Modify your approach",
					tc.Function.Name, err, tc.Function.Name, tc.Function.Arguments)
				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: tc.ID,
					Content:    errorMsg,
				})
			} else {
				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: tc.ID,
					Content:    toolOutput,
				})
			}
		}
	}
	return "", errors.New("exceeded maximum tool call iterations")
}

func (a *Agent) executeToolCall(toolCall openai.ToolCall, scriptMap map[string]string, skillPath string) (string, error) {
	var toolOutput string
	var err error

	switch toolCall.Function.Name {
	case "run_shell_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err = json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_script arguments: %w", err)
		}
		toolOutput, err = tool.RunShellScript(params.ScriptPath, params.Args)
	case "run_python_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err = json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_script arguments: %w", err)
		}
		toolOutput, err = tool.RunPythonScript(params.ScriptPath, params.Args)
	case "read_file":
		var params struct {
			FilePath string `json:"filePath"`
		}
		if err = json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal read_file arguments: %w", err)
		}
		path := params.FilePath
		if !filepath.IsAbs(path) && skillPath != "" {
			resolvedPath := filepath.Join(skillPath, path)
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
		if err = json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal write_file arguments: %w", err)
		}
		err = tool.WriteFile(params.FilePath, params.Content)
		if err == nil {
			toolOutput = fmt.Sprintf("Successfully wrote to file: %s", params.FilePath)
		}
	case "tavily_search":
		var params struct {
			Query string `json:"query"`
		}
		if err = json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal tavily_search arguments: %w", err)
		}
		toolOutput, err = tool.TavilySearch(params.Query)
	default:
		if scriptPath, ok := scriptMap[toolCall.Function.Name]; ok {
			var params struct {
				Args []string `json:"args"`
			}
			if toolCall.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
					return "", fmt.Errorf("failed to unmarshal script arguments: %w", err)
				}
			}
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
		log.Error("tool execution failed for %s: %v", toolCall.Function.Name, err)
		if toolCall.Function.Arguments != "" {
			log.Debug("raw arguments: %s", toolCall.Function.Arguments)
		}
		return "", fmt.Errorf("tool execution failed for %s: %w", toolCall.Function.Name, err)
	}
	return toolOutput, nil
}
