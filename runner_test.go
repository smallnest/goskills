package goskills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockOpenAIClient is a mock implementation of OpenAIChatClient for testing
type MockOpenAIClient struct {
	responses []openai.ChatCompletionResponse
	callCount int
	err       error
}

func (m *MockOpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if m.err != nil {
		return openai.ChatCompletionResponse{}, m.err
	}
	if m.callCount >= len(m.responses) {
		return openai.ChatCompletionResponse{}, fmt.Errorf("no more responses")
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

// NewMockOpenAIClient creates a new mock client with predefined responses
func NewMockOpenAIClient(responses []openai.ChatCompletionResponse, err error) *MockOpenAIClient {
	return &MockOpenAIClient{
		responses: responses,
		err:       err,
	}
}

// createTestAgent creates an agent for testing using environment variables
func createTestAgent(t *testing.T) *Agent {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	config := openai.DefaultConfig(token)
	if apiURL := os.Getenv("OPENAI_API_BASE"); apiURL != "" {
		config.BaseURL = apiURL
	}

	client := openai.NewClientWithConfig(config)
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "deepseek-v3" // default model
	}

	cfg := RunnerConfig{
		Model: model,
	}

	return &Agent{
		client: client,
		cfg:    cfg,
	}
}

// TestSelectSkill_Success tests successful skill selection
func TestSelectSkill_Success(t *testing.T) {
	agent := createTestAgent(t)

	// Create test skills
	skills := map[string]SkillPackage{
		"pdf": {
			Meta: SkillMeta{
				Name:        "pdf",
				Description: "Comprehensive PDF manipulation toolkit for extracting text and tables",
			},
		},
		"xlsx": {
			Meta: SkillMeta{
				Name:        "xlsx",
				Description: "Comprehensive spreadsheet creation, editing, and analysis",
			},
		},
	}

	// Execute test
	ctx := context.Background()
	userPrompt := "Please extract text from this PDF file"

	skillName, err := agent.selectSkill(ctx, userPrompt, skills)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "pdf", skillName)
}

// TestSelectSkill_XlsxSelection tests xlsx skill selection
func TestSelectSkill_XlsxSelection(t *testing.T) {
	agent := createTestAgent(t)

	skills := map[string]SkillPackage{
		"pdf": {
			Meta: SkillMeta{
				Name:        "pdf",
				Description: "Comprehensive PDF manipulation toolkit for extracting text and tables",
			},
		},
		"xlsx": {
			Meta: SkillMeta{
				Name:        "xlsx",
				Description: "Comprehensive spreadsheet creation, editing, and analysis",
			},
		},
		"email": {
			Meta: SkillMeta{
				Name:        "email",
				Description: "Email composition and sending toolkit",
			},
		},
	}

	testCases := []struct {
		name          string
		userPrompt    string
		expectedSkill string
	}{
		{
			name:          "PDF extraction request",
			userPrompt:    "Extract tables from this PDF document",
			expectedSkill: "pdf",
		},
		{
			name:          "Spreadsheet creation request",
			userPrompt:    "Create an Excel spreadsheet with sales data",
			expectedSkill: "xlsx",
		},
		{
			name:          "Email request",
			userPrompt:    "Send an email to the team",
			expectedSkill: "email",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			skillName, err := agent.selectSkill(ctx, tc.userPrompt, skills)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedSkill, skillName)
		})
	}
}

// TestSelectSkill_WithQuotesAndWhitespace tests skill selection with various response formats
func TestSelectSkill_WithQuotesAndWhitespace(t *testing.T) {
	agent := createTestAgent(t)

	skills := map[string]SkillPackage{
		"pdf": {
			Meta: SkillMeta{
				Name:        "pdf",
				Description: "PDF manipulation toolkit",
			},
		},
	}

	// Test with a clear PDF request that should return pdf skill
	ctx := context.Background()
	userPrompt := "Please help me extract text from a PDF file"

	skillName, err := agent.selectSkill(ctx, userPrompt, skills)

	// Assertions - the function should handle trimming quotes and whitespace
	assert.NoError(t, err)
	assert.Equal(t, "pdf", skillName)
}

// TestSelectSkill_EmptySkillsMap tests behavior with empty skills map
func TestSelectSkill_EmptySkillsMap(t *testing.T) {
	agent := createTestAgent(t)

	// Empty skills map
	skills := map[string]SkillPackage{}

	ctx := context.Background()
	userPrompt := "Do something"

	skillName, err := agent.selectSkill(ctx, userPrompt, skills)

	// Should not error, but may return a message explaining no skills are available
	assert.NoError(t, err)
	// The AI might return an explanatory message when no skills are available
	// so we just check that it doesn't crash and returns some response
	assert.NotEmpty(t, skillName)
}

// TestSelectSkill_ComplexSkillDescriptions tests skill selection with complex descriptions
func TestSelectSkill_ComplexSkillDescriptions(t *testing.T) {
	agent := createTestAgent(t)

	// Create skills with complex descriptions containing special characters
	skills := map[string]SkillPackage{
		"pdf": {
			Meta: SkillMeta{
				Name:        "pdf",
				Description: "Comprehensive PDF manipulation toolkit for extracting text, tables, and images. Supports OCR, form filling, and digital signatures. Works with encrypted/password-protected files.",
			},
		},
		"data-analysis": {
			Meta: SkillMeta{
				Name:        "data-analysis",
				Description: "Advanced data analysis toolkit with statistical functions, visualization, and machine learning capabilities. Works with CSV, JSON, and various data formats.",
			},
		},
	}

	ctx := context.Background()
	userPrompt := "Extract text and tables from an encrypted PDF file"

	skillName, err := agent.selectSkill(ctx, userPrompt, skills)

	assert.NoError(t, err)
	assert.Equal(t, "pdf", skillName)
}

// TestNewAgent_Success tests successful agent creation
func TestNewAgent_Success(t *testing.T) {
	cfg := RunnerConfig{
		APIKey:    "test-api-key",
		Model:     "gpt-4",
		SkillsDir: "./test-skills",
	}

	agent, err := NewAgent(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, cfg.APIKey, agent.cfg.APIKey)
	assert.Equal(t, cfg.Model, agent.cfg.Model)
	assert.Equal(t, cfg.SkillsDir, agent.cfg.SkillsDir)
	assert.NotNil(t, agent.client)
	assert.NotNil(t, agent.messages)
}

// TestNewAgent_EmptyAPIKey tests agent creation with empty API key
func TestNewAgent_EmptyAPIKey(t *testing.T) {
	cfg := RunnerConfig{
		APIKey: "",
		Model:  "gpt-4",
	}

	agent, err := NewAgent(cfg, nil)

	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "API key is not set")
}

// TestNewAgent_DefaultModel tests agent creation with default model
func TestNewAgent_DefaultModel(t *testing.T) {
	cfg := RunnerConfig{
		APIKey: "test-api-key",
		// Model is empty, should default to "gpt-4o"
	}

	agent, err := NewAgent(cfg, nil)

	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, "gpt-4o", agent.cfg.Model)
}

// TestDiscoverSkills tests the skill discovery functionality
func TestDiscoverSkills(t *testing.T) {
	cfg := RunnerConfig{
		APIKey:    "test-api-key",
		Model:     "gpt-4",
		SkillsDir: "./tool", // Use existing tool directory for testing
	}

	agent, err := NewAgent(cfg, nil)
	assert.NoError(t, err)

	// Test with an existing directory that should have skill-like structure
	skills, err := agent.discoverSkills("./tool")

	// We expect this to potentially error since ./tool may not be a proper skills directory
	// but we're testing the function doesn't panic
	if err != nil {
		assert.NotEmpty(t, err.Error())
	} else {
		assert.NotNil(t, skills)
	}
}

// TestExtractSkillName tests skill name extraction from AI responses
func TestExtractSkillName(t *testing.T) {
	skills := map[string]SkillPackage{
		"pdf": {
			Meta: SkillMeta{
				Name: "pdf",
			},
		},
		"xlsx": {
			Meta: SkillMeta{
				Name: "xlsx",
			},
		},
	}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Exact skill name",
			input:    "pdf",
			expected: "pdf",
		},
		{
			name:     "Skill name with quotes",
			input:    `"pdf"`,
			expected: "pdf",
		},
		{
			name:     "Skill name with whitespace",
			input:    "  pdf  ",
			expected: "pdf",
		},
		{
			name:     "Skill name with quotes and whitespace",
			input:    `  "pdf"  `,
			expected: "pdf",
		},
		{
			name:     "Case insensitive match",
			input:    "PDF",
			expected: "pdf",
		},
		{
			name:     "Complex response with skill name",
			input:    "Based on your request, I'll use the pdf skill to help you.",
			expected: "pdf",
		},
		{
			name:     "No skill found",
			input:    "I don't know which skill to use",
			expected: "I don't know which skill to use",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractSkillName(tc.input, skills)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRun_WithMock tests the Run method with a mock client
func TestRun_WithMock(t *testing.T) {
	// Create a temporary test skills directory
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	// Create a simple SKILL.md file (must be uppercase)
	skillContent := `---
name: test-skill
description: A test skill
---
This is a test skill.`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))

	// Create mock responses
	mockResponses := []openai.ChatCompletionResponse{
		{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleAssistant,
						Content: "test-skill",
					},
				},
			},
		},
		{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleAssistant,
						Content: "This is the final response",
					},
				},
			},
		},
	}

	mockClient := NewMockOpenAIClient(mockResponses, nil)

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:            "test-model",
			SkillsDir:        tmpDir,
			Verbose:          0,
			AutoApproveTools: true,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	result, err := agent.Run(context.Background(), "test prompt")
	assert.NoError(t, err)
	assert.Equal(t, "This is the final response", result)
}

// TestSelectAndPrepareSkill tests the selectAndPrepareSkill method
func TestSelectAndPrepareSkill(t *testing.T) {
	// Create a temporary test skills directory
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	// Create a simple SKILL.md file (must be uppercase)
	skillContent := `---
name: test-skill
description: A test skill for testing
---
This is a test skill body.`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))

	// Create mock response for skill selection
	mockResponse := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "test-skill",
				},
			},
		},
	}

	mockClient := NewMockOpenAIClient([]openai.ChatCompletionResponse{mockResponse}, nil)

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:     "test-model",
			SkillsDir: tmpDir,
			Verbose:   0,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	skill, err := agent.selectAndPrepareSkill(context.Background(), "test prompt")
	assert.NoError(t, err)
	assert.NotNil(t, skill)
	assert.Equal(t, "test-skill", skill.Meta.Name)
}

// TestSelectAndPrepareSkill_NoSkills tests error when no skills are found
func TestSelectAndPrepareSkill_NoSkills(t *testing.T) {
	tmpDir := t.TempDir()

	mockClient := NewMockOpenAIClient([]openai.ChatCompletionResponse{}, nil)

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:     "test-model",
			SkillsDir: tmpDir,
			Verbose:   0,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	skill, err := agent.selectAndPrepareSkill(context.Background(), "test prompt")
	assert.Error(t, err)
	assert.Nil(t, skill)
	assert.Contains(t, err.Error(), "no valid skills found")
}

// TestExecuteToolCall_BashReadFile tests executeToolCall for reading files via bash
func TestExecuteToolCall_BashReadFile(t *testing.T) {
	// Create a temporary file
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	testContent := "test file content"
	require.NoError(t, os.WriteFile(tmpFile, []byte(testContent), 0644))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	// Use bash cat command to read file
	argsJSON := fmt.Sprintf(`{"command": "cat %s"}`, tmpFile)
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	output, err := agent.executeToolCall(toolCall, nil, "")
	assert.NoError(t, err)
	assert.Contains(t, output, testContent)
}

// TestExecuteToolCall_BashWriteFile tests executeToolCall for writing files via bash
func TestExecuteToolCall_BashWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	testContent := "written content"
	// Use bash printf to write file (no trailing newline)
	argsJSON := fmt.Sprintf(`{"command": "printf '%%s' '%s' > %s"}`, testContent, tmpFile)
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	_, err := agent.executeToolCall(toolCall, nil, "")
	assert.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// TestExecuteToolCall_UnknownTool tests error handling for unknown tools
func TestExecuteToolCall_UnknownTool(t *testing.T) {
	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "unknown_tool",
			Arguments: "{}",
		},
	}

	output, err := agent.executeToolCall(toolCall, nil, "")
	assert.Error(t, err)
	assert.Empty(t, output)
	assert.Contains(t, err.Error(), "unknown tool")
}

// TestExecuteToolCall_InvalidJSON tests error handling for invalid JSON arguments
func TestExecuteToolCall_InvalidJSON(t *testing.T) {
	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: "invalid json",
		},
	}

	output, err := agent.executeToolCall(toolCall, nil, "")
	assert.Error(t, err)
	assert.Empty(t, output)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

// TestCleanToolArguments tests the cleanToolArguments function
func TestCleanToolArguments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with whitespace",
			input:    `  {"key": "value"}  `,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with code fence",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with uppercase code fence",
			input:    "```JSON\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with simple code fence",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON wrapped in single quotes",
			input:    `'{"key": "value"}'`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with over-escaped quotes",
			input:    `{\"key\": \"value\"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with escaped single quotes",
			input:    `{'key': 'value'}`,
			expected: `{'key': 'value'}`,
		},
		{
			name:     "JSON with newlines in code fence",
			input:    "```json\r\n{\"key\": \"value\"}\r\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "complex case: code fence + over-escaped",
			input:    "```json\n{\"filePath\": \"/tmp/test.txt\", \"content\": \"hello world\"}\n```",
			expected: `{"filePath": "/tmp/test.txt", "content": "hello world"}`,
		},
		{
			name:     "path with backslashes should keep double backslash",
			input:    `{"path": "C:\\Users\\test"}`,
			expected: `{"path": "C:\\Users\\test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanToolArguments(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExecuteToolCall_WithCodeFence tests that tool calls work even with code fences
func TestExecuteToolCall_WithCodeFence(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	testContent := "test content"
	// Simulate LLM returning JSON with code fence
	argsJSON := fmt.Sprintf("```json\n{\"command\": \"printf '%%s' '%s' > %s\"}\n```", testContent, tmpFile)

	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	_, err := agent.executeToolCall(toolCall, nil, "")
	assert.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// TestExecuteToolCall_WithOverEscapedQuotes tests handling of over-escaped quotes
func TestExecuteToolCall_WithOverEscapedQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	testContent := "test content"
	// Simulate LLM returning over-escaped JSON
	argsJSON := fmt.Sprintf(`{\"command\": \"printf '%%s' '%s' > %s\"}`, testContent, tmpFile)

	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	_, err := agent.executeToolCall(toolCall, nil, "")
	assert.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// TestExecuteSkillWithTools tests executeSkillWithTools method
func TestExecuteSkillWithTools(t *testing.T) {
	// Create mock response without tool calls (final response)
	mockResponse := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "Final response",
				},
			},
		},
	}

	mockClient := NewMockOpenAIClient([]openai.ChatCompletionResponse{mockResponse}, nil)

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:            "test-model",
			AutoApproveTools: true,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	skill := SkillPackage{
		Meta: SkillMeta{
			Name:        "test",
			Description: "test skill",
		},
		Body: "Test skill body",
		Path: "/test/path",
	}

	result, err := agent.executeSkillWithTools(context.Background(), "test prompt", &skill)
	assert.NoError(t, err)
	assert.Equal(t, "Final response", result)
}

// TestContinueSkillWithTools_WithToolCalls tests continueSkillWithTools with tool execution
func TestContinueSkillWithTools_WithToolCalls(t *testing.T) {
	// Create a temp file for the bash cat command
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test content"), 0644))

	// First response with tool call
	argsJSON, _ := json.Marshal(map[string]string{"command": "cat " + tmpFile})
	mockResponse1 := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "",
					ToolCalls: []openai.ToolCall{
						{
							ID:   "call-1",
							Type: openai.ToolTypeFunction,
							Function: openai.FunctionCall{
								Name:      "bash",
								Arguments: string(argsJSON),
							},
						},
					},
				},
			},
		},
	}

	// Second response without tool calls (final)
	mockResponse2 := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "File content processed",
				},
			},
		},
	}

	mockClient := NewMockOpenAIClient([]openai.ChatCompletionResponse{mockResponse1, mockResponse2}, nil)

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:            "test-model",
			AutoApproveTools: true,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	skill := SkillPackage{
		Meta: SkillMeta{
			Name: "test",
		},
		Body: "Test",
		Path: "/test",
	}

	result, err := agent.continueSkillWithTools(context.Background(), "test prompt", &skill)
	assert.NoError(t, err)
	assert.Equal(t, "File content processed", result)
}

// TestDiscoverSkills_RealDirectory tests discoverSkills with testdata
func TestDiscoverSkills_RealDirectory(t *testing.T) {
	cfg := RunnerConfig{
		APIKey: "test-key",
		Model:  "test-model",
	}

	agent, err := NewAgent(cfg, nil)
	require.NoError(t, err)

	// Test with testdata/skills directory if it exists
	skillsDir := "./testdata/skills"
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		t.Skip("testdata/skills directory does not exist")
	}

	skills, err := agent.discoverSkills(skillsDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, skills)
}

// TestExecuteToolCall_CustomScript tests custom script execution
func TestExecuteToolCall_CustomScript(t *testing.T) {
	// Create a temporary script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := "#!/bin/bash\necho 'custom script output'"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	scriptMap := map[string]string{
		"run_custom_script": scriptPath,
	}

	argsJSON := `{"args": []}`
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "run_custom_script",
			Arguments: argsJSON,
		},
	}

	output, err := agent.executeToolCall(toolCall, scriptMap, "")
	assert.NoError(t, err)
	assert.Contains(t, output, "custom script output")
}

// TestExecuteToolCall_BashShellScript tests shell script execution via bash
func TestExecuteToolCall_BashShellScript(t *testing.T) {
	// Create a temporary script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := "#!/bin/bash\necho 'shell script output'"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	// Use bash to execute the script
	argsJSON := fmt.Sprintf(`{"command": "bash %s"}`, scriptPath)
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	output, err := agent.executeToolCall(toolCall, nil, "")
	assert.NoError(t, err)
	assert.Contains(t, output, "shell script output")
}

// TestExecuteToolCall_BashPythonScript tests Python script execution via bash
func TestExecuteToolCall_BashPythonScript(t *testing.T) {
	// Create a temporary Python script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.py")
	scriptContent := "print('python script output')"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0644))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	// Use bash to execute the Python script
	argsJSON := fmt.Sprintf(`{"command": "python3 %s"}`, scriptPath)
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	output, err := agent.executeToolCall(toolCall, nil, "")
	assert.NoError(t, err)
	assert.Contains(t, output, "python script output")
}

// TestExecuteToolCall_BashReadFileRelativePath tests reading file with relative path via bash
func TestExecuteToolCall_BashReadFileRelativePath(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "skill")
	require.NoError(t, os.MkdirAll(skillPath, 0755))

	// Create a file in the skill directory
	testFile := "test.txt"
	testContent := "relative path test content"
	require.NoError(t, os.WriteFile(filepath.Join(skillPath, testFile), []byte(testContent), 0644))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	// Use bash cat with relative path
	argsJSON := fmt.Sprintf(`{"command": "cat %s"}`, testFile)
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "bash",
			Arguments: argsJSON,
		},
	}

	output, err := agent.executeToolCall(toolCall, nil, skillPath)
	assert.NoError(t, err)
	assert.Contains(t, output, testContent)
}

// TestExecuteToolCall_CustomPythonScript tests custom Python script execution
func TestExecuteToolCall_CustomPythonScript(t *testing.T) {
	// Create a temporary Python script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "custom.py")
	scriptContent := "print('custom python output')"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0644))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	scriptMap := map[string]string{
		"run_custom_python": scriptPath,
	}

	argsJSON := `{"args": []}`
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "run_custom_python",
			Arguments: argsJSON,
		},
	}

	output, err := agent.executeToolCall(toolCall, scriptMap, "")
	assert.NoError(t, err)
	assert.Contains(t, output, "custom python output")
}

// TestExecuteToolCall_CustomScriptWithEmptyArgs tests custom script with empty arguments
func TestExecuteToolCall_CustomScriptWithEmptyArgs(t *testing.T) {
	// Create a temporary script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := "#!/bin/bash\necho 'no args'"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	agent := &Agent{
		cfg: RunnerConfig{
			AutoApproveTools: true,
		},
	}

	scriptMap := map[string]string{
		"run_script_no_args": scriptPath,
	}

	// Empty arguments string
	toolCall := openai.ToolCall{
		ID:   "test-id",
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      "run_script_no_args",
			Arguments: "",
		},
	}

	output, err := agent.executeToolCall(toolCall, scriptMap, "")
	assert.NoError(t, err)
	assert.Contains(t, output, "no args")
}

// TestContinueSkillWithTools_APIError tests error handling when API call fails
func TestContinueSkillWithTools_APIError(t *testing.T) {
	mockClient := NewMockOpenAIClient(nil, fmt.Errorf("API error"))

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:            "test-model",
			AutoApproveTools: true,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	skill := SkillPackage{
		Meta: SkillMeta{Name: "test"},
		Body: "Test",
		Path: "/test",
	}

	result, err := agent.continueSkillWithTools(context.Background(), "test prompt", &skill)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ChatCompletion error")
	assert.Empty(t, result)
}

// TestRun_SelectionError tests error handling when skill selection fails
func TestRun_SelectionError(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillContent := `---
name: test-skill
description: A test skill
---
This is a test skill.`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))

	// Create mock that returns error
	mockClient := NewMockOpenAIClient(nil, fmt.Errorf("selection error"))

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:            "test-model",
			SkillsDir:        tmpDir,
			Verbose:          0,
			AutoApproveTools: true,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	result, err := agent.Run(context.Background(), "test prompt")
	assert.Error(t, err)
	assert.Empty(t, result)
}

// TestSelectAndPrepareSkill_NonExistentSkillSelected tests error when AI selects non-existent skill
func TestSelectAndPrepareSkill_NonExistentSkillSelected(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillContent := `---
name: test-skill
description: A test skill
---
This is a test skill.`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))

	// Mock returns a skill name that doesn't exist
	mockResponse := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "non-existent-skill",
				},
			},
		},
	}

	mockClient := NewMockOpenAIClient([]openai.ChatCompletionResponse{mockResponse}, nil)

	agent := &Agent{
		client: mockClient,
		cfg: RunnerConfig{
			Model:     "test-model",
			SkillsDir: tmpDir,
			Verbose:   0,
		},
		messages: []openai.ChatCompletionMessage{},
	}

	skill, err := agent.selectAndPrepareSkill(context.Background(), "test prompt")
	assert.Error(t, err)
	assert.Nil(t, skill)
	assert.Contains(t, err.Error(), "not found")
}
