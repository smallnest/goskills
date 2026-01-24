package goskills

import (
	"fmt"
	"path/filepath"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/goskills/tool"
)

// GenerateToolDefinitions generates the list of OpenAI tools for a given skill.
// It returns the tool definitions and a map of tool names to script paths for execution.
// 优先使用 SKILL.md 中定义的工具，如果未定义则自动生成。
func GenerateToolDefinitions(skill *SkillPackage) ([]openai.Tool, map[string]string) {
	var tools []openai.Tool
	scriptMap := make(map[string]string)

	// 1. Base Tools
	baseTools := tool.GetBaseTools()

	if len(skill.Meta.AllowedTools) > 0 {
		allowedMap := make(map[string]bool)
		for _, t := range skill.Meta.AllowedTools {
			allowedMap[t] = true
		}

		for _, t := range baseTools {
			if allowedMap[t.Function.Name] {
				tools = append(tools, t)
			}
		}
	} else {
		tools = append(tools, baseTools...)
	}

	// 2. Script Tools - 优先使用 SKILL.md 中定义的工具
	if len(skill.Meta.Tools) > 0 {
		// 使用 SKILL.md 中定义的工具
		for _, toolDef := range skill.Meta.Tools {
			tool, scriptPath := generateToolFromDefinition(skill.Path, skill.Resources.Scripts, toolDef)
			tools = append(tools, tool)
			if scriptPath != "" {
				scriptMap[tool.Function.Name] = scriptPath
			}
		}
	} else {
		// 自动生成脚本工具
		for _, scriptRelPath := range skill.Resources.Scripts {
			toolDef, toolName := generateScriptTool(skill.Path, scriptRelPath)
			tools = append(tools, toolDef)
			scriptMap[toolName] = filepath.Join(skill.Path, scriptRelPath)
		}
	}

	return tools, scriptMap
}

// generateToolFromDefinition 从 SKILL.md 中的工具定义生成 OpenAI 工具
func generateToolFromDefinition(skillPath string, scripts []string, toolDef ToolDefinition) (openai.Tool, string) {
	// 确定脚本路径
	var scriptPath string
	if toolDef.Script != "" {
		scriptPath = filepath.Join(skillPath, toolDef.Script)
	} else {
		// 尝试从工具名推断脚本路径
		// 例如: generate_comic_storyboard -> scripts/generate-comic.ts
		scriptName := strings.ReplaceAll(toolDef.Name, "_", "-")
		for _, ext := range []string{".ts", ".js", ".py", ".sh"} {
			candidatePath := filepath.Join(skillPath, "scripts", scriptName+ext)
			// 检查文件是否存在（简单检查）
			for _, script := range scripts {
				fullScriptPath := filepath.Join(skillPath, script)
				if fullScriptPath == candidatePath {
					scriptPath = candidatePath
					break
				}
			}
			if scriptPath != "" {
				break
			}
		}
	}

	// 构建参数 schema
	parameters := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}

	if len(toolDef.Parameters) > 0 {
		var required []string
		for paramName, param := range toolDef.Parameters {
			prop := map[string]any{
				"type": param.Type,
			}
			if param.Description != "" {
				prop["description"] = param.Description
			}
			parameters["properties"].(map[string]any)[paramName] = prop
			if param.Required {
				required = append(required, paramName)
			}
		}
		if len(required) > 0 {
			parameters["required"] = required
		}
	} else {
		// 默认参数: args 数组
		parameters["properties"] = map[string]any{
			"args": map[string]any{
				"type":        "array",
				"description": "Arguments to pass to the script.",
				"items": map[string]any{
					"type": "string",
				},
			},
		}
	}

	description := toolDef.Description
	if description == "" {
		description = fmt.Sprintf("Executes tool '%s'", toolDef.Name)
	}

	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        toolDef.Name,
			Description: description,
			Parameters:  parameters,
		},
	}, scriptPath
}

// GetToolDefinitions 返回 skill 中定义的工具列表（用于外部访问）
func GetToolDefinitions(skill *SkillPackage) []ToolDefinition {
	return skill.Meta.Tools
}

func generateScriptTool(skillPath, scriptRelPath string) (openai.Tool, string) {
	// Normalize name: replace non-alphanumeric with underscore
	safeName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, scriptRelPath)
	toolName := "run_" + safeName

	// Determine type based on extension
	ext := filepath.Ext(scriptRelPath)
	var description string
	if ext == ".py" {
		description = fmt.Sprintf("Executes the python script '%s'.", scriptRelPath)
	} else {
		description = fmt.Sprintf("Executes the shell script '%s'.", scriptRelPath)
	}

	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        toolName,
			Description: description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"args": map[string]any{
						"type":        "array",
						"description": "Arguments to pass to the script.",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}, toolName
}
