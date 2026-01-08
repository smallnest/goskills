package tool

import (
	"encoding/json"
	"reflect"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestGetBaseTools(t *testing.T) {
	tools := GetBaseTools()

	// Test that we get the expected number of tools
	expectedCount := 8 // Based on the current implementation
	if len(tools) != expectedCount {
		t.Errorf("GetBaseTools() returned %d tools, expected %d", len(tools), expectedCount)
	}

	// Test that all tools are of type function
	for i, tool := range tools {
		if tool.Type != openai.ToolTypeFunction {
			t.Errorf("Tool %d has type %s, expected %s", i, tool.Type, openai.ToolTypeFunction)
		}

		if tool.Function == nil {
			t.Errorf("Tool %d has nil Function", i)
			continue
		}

		// Test that each function has required fields
		if tool.Function.Name == "" {
			t.Errorf("Tool %d has empty Name", i)
		}

		if tool.Function.Description == "" {
			t.Errorf("Tool %d has empty Description", i)
		}

		// Test that parameters have the expected structure
		if tool.Function.Parameters == nil {
			t.Errorf("Tool %d has nil Parameters", i)
			continue
		}

		params, ok := tool.Function.Parameters.(map[string]any)
		if !ok {
			t.Errorf("Tool %d parameters are not a map[string]interface{}", i)
			continue
		}

		// Check required fields in parameters
		if params["type"] != "object" {
			t.Errorf("Tool %d parameters type is not 'object'", i)
		}

		properties, ok := params["properties"].(map[string]any)
		if !ok {
			t.Errorf("Tool %d parameters properties are not a map[string]interface{}", i)
			continue
		}

		// Each tool should have at least one property
		if len(properties) == 0 {
			t.Errorf("Tool %d has no properties defined", i)
		}
	}
}

func TestGetBaseToolsSpecificTools(t *testing.T) {
	tools := GetBaseTools()
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Function.Name] = true
	}

	// Test that expected tools exist
	expectedTools := []string{
		"run_shell_script",
		"run_python_script",
		"read_file",
		"write_file",
		"tavily_search",
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found in GetBaseTools()", expectedTool)
		}
	}
}

func TestToolDefinitionsStructure(t *testing.T) {
	tools := GetBaseTools()

	// Test specific tool definitions
	testCases := []struct {
		name           string
		expectedDesc   string
		expectedParams []string
		requiredParams []string
	}{
		{
			name:           "run_shell_script",
			expectedDesc:   "Executes a shell script and returns its combined stdout and stderr. Use this for general shell commands.",
			expectedParams: []string{"scriptPath", "args"},
			requiredParams: []string{"scriptPath"},
		},
		{
			name:           "run_python_script",
			expectedDesc:   "Executes a Python script and returns its combined stdout and stderr.",
			expectedParams: []string{"scriptPath", "args"},
			requiredParams: []string{"scriptPath"},
		},
		{
			name:           "read_file",
			expectedDesc:   "Reads the content of a file and returns it as a string.",
			expectedParams: []string{"filePath"},
			requiredParams: []string{"filePath"},
		},
		{
			name:           "write_file",
			expectedDesc:   "Writes the given content to a file. If the file does not exist, it will be created. If it exists, its content will be truncated.",
			expectedParams: []string{"filePath", "content"},
			requiredParams: []string{"filePath", "content"},
		},
		{
			name:           "tavily_search",
			expectedDesc:   "Performs a web search using the Tavily API for the given query and returns a summary of results.",
			expectedParams: []string{"query"},
			requiredParams: []string{"query"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var foundTool *openai.FunctionDefinition
			for _, tool := range tools {
				if tool.Function.Name == tc.name {
					foundTool = tool.Function
					break
				}
			}

			if foundTool == nil {
				t.Errorf("Tool %s not found", tc.name)
				return
			}

			// Check description
			if foundTool.Description != tc.expectedDesc {
				t.Errorf("Tool %s description = %q, expected %q", tc.name, foundTool.Description, tc.expectedDesc)
			}

			// Check parameters
			params, ok := foundTool.Parameters.(map[string]any)
			if !ok {
				t.Errorf("Tool %s parameters are not in expected format", tc.name)
				return
			}

			properties, ok := params["properties"].(map[string]any)
			if !ok {
				t.Errorf("Tool %s properties are not in expected format", tc.name)
				return
			}

			// Check expected properties exist
			for _, expectedParam := range tc.expectedParams {
				if _, exists := properties[expectedParam]; !exists {
					t.Errorf("Tool %s missing expected parameter %s", tc.name, expectedParam)
				}
			}

			// Check required parameters (may be nil if no required params)
			if required, exists := params["required"]; exists {
				requiredSlice, ok := required.([]string)
				if !ok {
					// Try to convert from []interface{} to []string
					if ifaceSlice, ok := required.([]any); ok {
						requiredSlice = make([]string, len(ifaceSlice))
						for i, r := range ifaceSlice {
							if s, ok := r.(string); ok {
								requiredSlice[i] = s
							}
						}
					} else {
						t.Errorf("Tool %s required parameters are not in expected format", tc.name)
						return
					}
				}

				if !reflect.DeepEqual(requiredSlice, tc.requiredParams) {
					t.Errorf("Tool %s required parameters = %v, expected %v", tc.name, requiredSlice, tc.requiredParams)
				}
			}
		})
	}
}

func TestToolParameterTypes(t *testing.T) {
	tools := GetBaseTools()

	// Test that parameters have correct types
	testCases := []struct {
		toolName     string
		paramName    string
		expectedType string
		expectedDesc string
	}{
		{
			toolName:     "run_shell_code",
			paramName:    "code",
			expectedType: "string",
			expectedDesc: "The shell code snippet to execute.",
		},
		{
			toolName:     "run_shell_code",
			paramName:    "args",
			expectedType: "object",
			expectedDesc: "A map of key-value pairs to pass to the code.",
		},
		{
			toolName:     "run_shell_script",
			paramName:    "scriptPath",
			expectedType: "string",
			expectedDesc: "The path to the shell script to execute.",
		},
		{
			toolName:     "run_shell_script",
			paramName:    "args",
			expectedType: "array",
			expectedDesc: "A list of string arguments to pass to the script.",
		},
		{
			toolName:     "write_file",
			paramName:    "content",
			expectedType: "string",
			expectedDesc: "The content to write to the file.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.toolName+"_"+tc.paramName, func(t *testing.T) {
			var foundTool *openai.FunctionDefinition
			for _, tool := range tools {
				if tool.Function.Name == tc.toolName {
					foundTool = tool.Function
					break
				}
			}

			if foundTool == nil {
				t.Errorf("Tool %s not found", tc.toolName)
				return
			}

			params, ok := foundTool.Parameters.(map[string]any)
			if !ok {
				t.Errorf("Tool %s parameters are not in expected format", tc.toolName)
				return
			}

			properties, ok := params["properties"].(map[string]any)
			if !ok {
				t.Errorf("Tool %s properties are not in expected format", tc.toolName)
				return
			}

			param, exists := properties[tc.paramName]
			if !exists {
				t.Errorf("Tool %s missing parameter %s", tc.toolName, tc.paramName)
				return
			}

			paramMap, ok := param.(map[string]any)
			if !ok {
				t.Errorf("Tool %s parameter %s is not in expected format", tc.toolName, tc.paramName)
				return
			}

			// Check type
			if paramMap["type"] != tc.expectedType {
				t.Errorf("Tool %s parameter %s type = %v, expected %s", tc.toolName, tc.paramName, paramMap["type"], tc.expectedType)
			}

			// Check description
			if paramMap["description"] != tc.expectedDesc {
				t.Errorf("Tool %s parameter %s description = %v, expected %s", tc.toolName, tc.paramName, paramMap["description"], tc.expectedDesc)
			}
		})
	}
}

func TestToolsJSONSerialization(t *testing.T) {
	// Test that the tools can be properly serialized to JSON
	tools := GetBaseTools()

	// Convert to JSON
	jsonData, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal tools to JSON: %v", err)
		return
	}

	// Convert back from JSON
	var unmarshaledTools []openai.Tool
	err = json.Unmarshal(jsonData, &unmarshaledTools)
	if err != nil {
		t.Errorf("Failed to unmarshal tools from JSON: %v", err)
		return
	}

	// Compare original and unmarshaled tools
	if len(unmarshaledTools) != len(tools) {
		t.Errorf("JSON round trip changed tool count from %d to %d", len(tools), len(unmarshaledTools))
	}

	for i, tool := range tools {
		if i >= len(unmarshaledTools) {
			t.Errorf("Missing tool %d after JSON round trip", i)
			continue
		}

		unmarshaledTool := unmarshaledTools[i]

		if tool.Type != unmarshaledTool.Type {
			t.Errorf("Tool %d type changed from %s to %s", i, tool.Type, unmarshaledTool.Type)
		}

		if tool.Function.Name != unmarshaledTool.Function.Name {
			t.Errorf("Tool %d name changed from %s to %s", i, tool.Function.Name, unmarshaledTool.Function.Name)
		}

		if tool.Function.Description != unmarshaledTool.Function.Description {
			t.Errorf("Tool %d description changed from %s to %s", i, tool.Function.Description, unmarshaledTool.Function.Description)
		}
	}
}

func BenchmarkGetBaseTools(b *testing.B) {
	for b.Loop() {
		_ = GetBaseTools()
	}
}
