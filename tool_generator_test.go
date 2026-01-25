package goskills

import (
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

// TestGenerateToolDefinitions_AllowedTools tests tool generation with allowed tools filter
func TestGenerateToolDefinitions_AllowedTools(t *testing.T) {
	skill := SkillPackage{
		Path: "/test/skill",
		Meta: SkillMeta{
			AllowedTools: []string{"bash"}, // Only allow bash tool
		},
		Resources: SkillResources{
			Scripts: []string{"test.py", "setup.sh"},
		},
	}

	tools, scriptMap := GenerateToolDefinitions(&skill)

	// Should have 1 allowed base tool + 2 script tools
	assert.Len(t, tools, 3)
	assert.Len(t, scriptMap, 2)

	// Check that only allowed tools are included
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Function.Name] = true
	}
	assert.True(t, toolNames["bash"])
	assert.True(t, toolNames["run_test_py"])
	assert.True(t, toolNames["run_setup_sh"])
}

// TestGenerateToolDefinitions_NoAllowedTools tests tool generation without allowed tools filter
func TestGenerateToolDefinitions_NoAllowedTools(t *testing.T) {
	skill := SkillPackage{
		Path: "/test/skill",
		Meta: SkillMeta{
			AllowedTools: []string{}, // Empty allowed tools means all tools
		},
		Resources: SkillResources{
			Scripts: []string{"deploy.sh"},
		},
	}

	tools, scriptMap := GenerateToolDefinitions(&skill)

	// Should have all base tools (bash, tavily_search) + 1 script tool
	assert.GreaterOrEqual(t, len(tools), 3) // At least 2 base tools + 1 script
	assert.Len(t, scriptMap, 1)             // One script tool

	// Check script map contains correct path
	assert.Contains(t, scriptMap, "run_deploy_sh")
	assert.Equal(t, "/test/skill/deploy.sh", scriptMap["run_deploy_sh"])
}

// TestGenerateToolDefinitions_AllBaseTools tests that all base tools are included when no filter
func TestGenerateToolDefinitions_AllBaseTools(t *testing.T) {
	skill := SkillPackage{
		Path: "/test/skill",
		Meta: SkillMeta{
			AllowedTools: []string{}, // No filter - all tools
		},
		Resources: SkillResources{
			Scripts: []string{}, // No scripts
		},
	}

	tools, scriptMap := GenerateToolDefinitions(&skill)

	// Should have exactly 2 base tools (bash, tavily_search)
	assert.Len(t, tools, 2)
	assert.Len(t, scriptMap, 0) // No script tools

	// Check base tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Function.Name] = true
	}
	assert.True(t, toolNames["bash"])
	assert.True(t, toolNames["tavily_search"])
}

// TestGenerateToolDefinitions_OnlyBash tests filtering to only bash
func TestGenerateToolDefinitions_OnlyBash(t *testing.T) {
	skill := SkillPackage{
		Path: "/test/skill",
		Meta: SkillMeta{
			AllowedTools: []string{"bash"},
		},
		Resources: SkillResources{
			Scripts: []string{}, // No scripts
		},
	}

	tools, scriptMap := GenerateToolDefinitions(&skill)

	// Should have only bash tool
	assert.Len(t, tools, 1)
	assert.Len(t, scriptMap, 0)

	// Check it's the bash tool
	assert.Equal(t, "bash", tools[0].Function.Name)
	assert.Contains(t, tools[0].Function.Description, "shell command")
}

// TestGenerateScriptTool_PythonScript tests Python script tool generation
func TestGenerateScriptTool_PythonScript(t *testing.T) {
	skillPath := "/test/skill"
	scriptRelPath := "scripts/test.py"

	tool, toolName := generateScriptTool(skillPath, scriptRelPath)

	// Check tool name generation
	assert.Equal(t, "run_scripts_test_py", toolName)

	// Check tool definition
	assert.Equal(t, openai.ToolTypeFunction, tool.Type)
	assert.Equal(t, toolName, tool.Function.Name)
	assert.Contains(t, tool.Function.Description, "python script")
	assert.Contains(t, tool.Function.Description, "scripts/test.py")

	// Verify tool structure
	assert.NotNil(t, tool.Function)
	assert.NotNil(t, tool.Function.Parameters)
}

// TestGenerateScriptTool_ShellScript tests shell script tool generation
func TestGenerateScriptTool_ShellScript(t *testing.T) {
	skillPath := "/test/skill"
	scriptRelPath := "deploy.sh"

	tool, toolName := generateScriptTool(skillPath, scriptRelPath)

	// Check tool name generation
	assert.Equal(t, "run_deploy_sh", toolName)

	// Check tool definition
	assert.Equal(t, openai.ToolTypeFunction, tool.Type)
	assert.Equal(t, toolName, tool.Function.Name)
	assert.Contains(t, tool.Function.Description, "shell script")
	assert.Contains(t, tool.Function.Description, "deploy.sh")

	// Verify tool structure
	assert.NotNil(t, tool.Function)
	assert.NotNil(t, tool.Function.Parameters)
}

// TestGenerateScriptTool_TypeScriptScript tests TypeScript script tool generation
func TestGenerateScriptTool_TypeScriptScript(t *testing.T) {
	skillPath := "/test/skill"
	scriptRelPath := "build.ts"

	tool, toolName := generateScriptTool(skillPath, scriptRelPath)

	// Check tool name generation
	assert.Equal(t, "run_build_ts", toolName)

	// Check tool definition
	assert.Equal(t, openai.ToolTypeFunction, tool.Type)
	assert.Equal(t, toolName, tool.Function.Name)
	assert.Contains(t, tool.Function.Description, "shell script")
	assert.Contains(t, tool.Function.Description, "build.ts")

	// Verify tool structure
	assert.NotNil(t, tool.Function)
	assert.NotNil(t, tool.Function.Parameters)
}

// TestGenerateScriptTool_SpecialCharacters tests script tool generation with special characters
func TestGenerateScriptTool_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		scriptRelPath string
		expectedName  string
	}{
		{
			scriptRelPath: "my-script.sh",
			expectedName:  "run_my_script_sh",
		},
		{
			scriptRelPath: "test script.py",
			expectedName:  "run_test_script_py",
		},
		{
			scriptRelPath: "setup-v1.0.sh",
			expectedName:  "run_setup_v1_0_sh",
		},
		{
			scriptRelPath: "data@process.py",
			expectedName:  "run_data_process_py",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.scriptRelPath, func(t *testing.T) {
			tool, toolName := generateScriptTool("/test", tc.scriptRelPath)
			assert.Equal(t, tc.expectedName, toolName)
			assert.NotNil(t, tool.Function)
		})
	}
}

// TestGenerateToolDefinitions_EmptySkill tests tool generation with minimal skill
func TestGenerateToolDefinitions_EmptySkill(t *testing.T) {
	skill := SkillPackage{
		Path: "/test/skill",
		Meta: SkillMeta{}, // No allowed tools specified
		Resources: SkillResources{
			Scripts: []string{}, // No scripts
		},
	}

	tools, scriptMap := GenerateToolDefinitions(&skill)

	// Should have all base tools (bash, tavily_search)
	assert.Greater(t, len(tools), 0)
	assert.Len(t, scriptMap, 0)

	// Verify tool structure
	for _, tool := range tools {
		assert.Equal(t, openai.ToolTypeFunction, tool.Type)
		assert.NotNil(t, tool.Function)
		assert.NotEmpty(t, tool.Function.Name)
	}
}

// TestGenerateScriptTool_ParametersStructure tests that parameters structure is correct
func TestGenerateScriptTool_ParametersStructure(t *testing.T) {
	skillPath := "/test/skill"
	scriptRelPath := "test.py"

	tool, _ := generateScriptTool(skillPath, scriptRelPath)

	// Check that parameters is a map
	assert.IsType(t, map[string]any{}, tool.Function.Parameters)

	params := tool.Function.Parameters.(map[string]any)
	assert.Equal(t, "object", params["type"])
	assert.Contains(t, params, "properties")

	// Check properties structure
	properties := params["properties"].(map[string]any)
	assert.Contains(t, properties, "args")

	args := properties["args"].(map[string]any)
	assert.Equal(t, "array", args["type"])
	assert.Equal(t, "Arguments to pass to the script.", args["description"])
}
