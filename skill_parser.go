package goskills

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillPackage represents a fully and finely parsed Claude Skill package
type SkillPackage struct {
	Path      string         `json:"path"`
	Meta      SkillMeta      `json:"meta"`
	Body      string         `json:"body"` // Raw Markdown content of SKILL.md body
	Resources SkillResources `json:"resources"`
}

// ToolParameter 定义工具参数
type ToolParameter struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
}

// ToolDefinition 定义单个工具的配置
type ToolDefinition struct {
	Name        string                    `yaml:"name"`
	Script      string                    `yaml:"script,omitempty"` // 可选，指定脚本路径
	Description string                    `yaml:"description,omitempty"`
	Parameters  map[string]ToolParameter  `yaml:"parameters,omitempty"`
}

// SkillMeta corresponds to the content of SKILL.md frontmatter
type SkillMeta struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	AllowedTools []string          `yaml:"allowed-tools"`
	Model        string            `yaml:"model,omitempty"`
	Author       string            `yaml:"author,omitempty"`
	Version      string            `yaml:"version,omitempty"`
	License      string            `yaml:"license,omitempty"`
	Tools        []ToolDefinition  `yaml:"tools,omitempty"` // 工具定义列表
}

// SkillResources lists the relevant resource files in the skill package
type SkillResources struct {
	Scripts    []string `json:"scripts"`
	References []string `json:"references"`
	Assets     []string `json:"assets"`
	Templates  []string `json:"templates"`
}

// extractFrontmatterAndBody separates and parses the frontmatter and body of SKILL.md
func extractFrontmatterAndBody(data []byte) (SkillMeta, string, error) {
	marker := []byte("---")
	var meta SkillMeta
	var body string

	// Check if content starts with frontmatter marker
	content := string(data)
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return meta, "", fmt.Errorf("no YAML frontmatter found or format is incorrect")
	}

	parts := bytes.SplitN(data, marker, 3)
	if len(parts) < 3 {
		return meta, "", fmt.Errorf("no YAML frontmatter found or format is incorrect")
	}

	// Parse frontmatter
	if err := yaml.Unmarshal(parts[1], &meta); err != nil {
		return meta, "", fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
	}

	// Extract body
	body = strings.TrimSpace(string(parts[2]))

	return meta, body, nil
}

// parseOpenAISkill parses an OpenAI skill.md file without frontmatter
// The skill name comes from the directory name
// The description is extracted from between the first # heading and the first ## heading
func parseOpenAISkill(skillDir string, data []byte) (SkillMeta, string, error) {
	content := string(data)
	var meta SkillMeta
	var body string

	// Extract skill name from directory path
	dirName := filepath.Base(skillDir)
	meta.Name = strings.ReplaceAll(dirName, "-", " ")
	meta.Name = strings.ReplaceAll(meta.Name, "_", " ")
	// Don't convert to singular for OpenAI skills, as directory names are already proper

	// Use regex to find description between first # heading and first ## heading
	// Pattern: content after first # heading until before first ## heading
	descRegex := regexp.MustCompile(`(?s)^#\s+.*?\n\n(.*?)\n##`)
	matches := descRegex.FindStringSubmatch(content)

	if len(matches) > 1 {
		// Clean up the description
		description := strings.TrimSpace(matches[1])
		// Remove extra whitespace and newlines
		description = regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")
		meta.Description = description
	} else {
		// Fallback: extract first paragraph after the first # heading
		lines := strings.Split(content, "\n")
		inFirstSection := false
		var descLines []string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") && !inFirstSection {
				inFirstSection = true
				continue
			}
			if inFirstSection {
				if strings.HasPrefix(line, "##") || strings.HasPrefix(line, "# ") {
					break
				}
				if line != "" {
					descLines = append(descLines, line)
				}
			}
		}

		if len(descLines) > 0 {
			meta.Description = strings.Join(descLines, " ")
		} else {
			meta.Description = meta.Name
		}
	}

	// Prepend environment mapping information for OpenAI skills
	envMapping := `## 工具使用
你需要搜索相应的工具使用方法决定如何使用工具：
- 基于你的历史经验
- 搜索工具的官方文档
- 查看工具的help信息

## Original Skill Content

` + content

	// The modified content with environment mappings
	body = envMapping

	return meta, body, nil
}

// findResourceFiles finds all files in the specified resource directory
func findResourceFiles(skillPath, resourceDir string) ([]string, error) {
	var files []string
	scanDir := filepath.Join(skillPath, resourceDir)

	// Check if directory exists
	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		return files, nil // Directory does not exist, return empty list, no error
	}

	err := filepath.WalkDir(scanDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// Record path relative to the skill root directory
			relPath, err := filepath.Rel(skillPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})

	return files, err
}

// ParseSkillPackage finely parses the Skill package in the given directory path
func ParseSkillPackage(dirPath string) (*SkillPackage, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("skill directory not found: %s", dirPath)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// 1. Parse skill file - try both SKILL.md (Claude) and skill.md (OpenAI)
	var meta SkillMeta
	var bodyStr string
	var mdContent []byte

	// Check what files actually exist (to handle case-insensitive filesystems)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill directory: %w", err)
	}

	hasClaudeSkill := false
	hasOpenAISkill := false

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if name == "SKILL.md" {
				hasClaudeSkill = true
			} else if name == "skill.md" {
				hasOpenAISkill = true
			}
		}
	}

	if hasClaudeSkill {
		// Claude skill format with frontmatter
		skillMdPath := filepath.Join(dirPath, "SKILL.md")
		mdContent, err = os.ReadFile(skillMdPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
		}
		meta, bodyStr, err = extractFrontmatterAndBody(mdContent)
		if err != nil {
			return nil, err
		}
	} else if hasOpenAISkill {
		// OpenAI skill format without frontmatter
		skillMdPath := filepath.Join(dirPath, "skill.md")
		mdContent, err = os.ReadFile(skillMdPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read skill.md: %w", err)
		}
		meta, bodyStr, err = parseOpenAISkill(dirPath, mdContent)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("neither SKILL.md nor skill.md found in skill directory: %s", dirPath)
	}

	// 2. Find resource files
	scripts, err := findResourceFiles(dirPath, "scripts")
	if err != nil {
		return nil, fmt.Errorf("error scanning 'scripts' directory: %w", err)
	}
	references, err := findResourceFiles(dirPath, "references")
	if err != nil {
		return nil, fmt.Errorf("error scanning 'references' directory: %w", err)
	}
	assets, err := findResourceFiles(dirPath, "assets")
	if err != nil {
		return nil, fmt.Errorf("error scanning 'assets' directory: %w", err)
	}
	templates, err := findResourceFiles(dirPath, "templates")
	if err != nil {
		return nil, fmt.Errorf("error scanning 'templates' directory: %w", err)
	}

	// 3. Assemble SkillPackage
	pkg := &SkillPackage{
		Path: dirPath,
		Meta: meta,
		Body: bodyStr, // Store raw markdown body
		Resources: SkillResources{
			Scripts:    scripts,
			References: references,
			Assets:     assets,
			Templates:  templates,
		},
	}

	return pkg, nil
}

// ParseSkillPackages finds all skill packages in a given directory and its subdirectories.
// A directory is considered a skill package if it contains either a SKILL.md (Claude) or skill.md (OpenAI) file.
// It returns a slice of successfully parsed SkillPackage objects.

func ParseSkillPackages(rootDir string) ([]*SkillPackage, error) {
	skillDirs := make(map[string]struct{})

	walkErr := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && (d.Name() == "SKILL.md" || d.Name() == "skill.md") {
			dir := filepath.Dir(path)
			skillDirs[dir] = struct{}{}
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", rootDir, walkErr)
	}

	var packages []*SkillPackage
	for dir := range skillDirs {
		pkg, err := ParseSkillPackage(dir)
		if err == nil {
			packages = append(packages, pkg)
		}

		// Silently ignore packages that fail to parse
	}

	return packages, nil
}

// SkillsToPrompt converts a slice of SkillPackage objects to a prompt string
func SkillsToPrompt(skills map[string]SkillPackage) string {
	var builder strings.Builder

	// Add skills instructions header
	builder.WriteString("<skills_instructions>\n")
	builder.WriteString("When users ask you to perform tasks, check if any of the available skills below can help complete the task more effectively.\n\n")

	builder.WriteString("How to use skills:\n")
	builder.WriteString("- Invoke skills using this tool with the skill name only (no arguments)\n")
	builder.WriteString("- When you invoke a skill, you will see <command-message>The \"{name}\" skill is loading</command-message>\n")
	builder.WriteString("- The skill's prompt will expand and provide detailed instructions on how to complete the task\n\n")

	builder.WriteString("Important:\n")
	builder.WriteString("- Only use skills listed in <available_skills> below\n")
	builder.WriteString("- Do not invoke a skill that is already running\n")
	builder.WriteString("</skills_instructions>\n\n")

	// Add available tools instructions
	builder.WriteString("<available_tools_instructions>\n")
	builder.WriteString("When working on tasks, you have access to the following tools:\n\n")
	builder.WriteString("**File Operations:**\n")
	builder.WriteString("- `write_file(filePath, content)`: Create or overwrite a file with the given content\n")
	builder.WriteString("- `read_file(filePath)`: Read the contents of a file\n")
	builder.WriteString("  Use these when you need to create, modify, or read files as part of your task.\n\n")

	builder.WriteString("**Script Execution:**\n")
	builder.WriteString("- `run_shell_script(scriptPath, args)`: Execute a shell script\n")
	builder.WriteString("- `run_python_script(scriptPath, args)`: Execute a Python script\n")
	builder.WriteString("  Use these when the task requires running scripts in the skill's scripts directory.\n\n")

	builder.WriteString("**IMPORTANT Guidelines:**\n")
	builder.WriteString("- When a skill asks you to \"create a file\" or \"write to a file\", you MUST call the `write_file` tool\n")
	builder.WriteString("- When a skill asks you to \"read a file\", you MUST call the `read_file` tool\n")
	builder.WriteString("- Do NOT just output file contents - actually call the tools to create/read files\n")
	builder.WriteString("- Only execute scripts that are part of the skill's scripts directory\n")
	builder.WriteString("</available_tools_instructions>\n\n")

	// Add available skills section
	builder.WriteString("<available_skills>\n")

	for _, skill := range skills {
		builder.WriteString("<skill>\n")
		builder.WriteString(fmt.Sprintf("<name>%s</name>\n", skill.Meta.Name))
		builder.WriteString(fmt.Sprintf("<description>%s</description>\n", skill.Meta.Description))
		builder.WriteString("<location>plugin</location>\n")
		builder.WriteString("</skill>\n\n")
	}

	builder.WriteString("</available_skills>")

	return builder.String()
}
