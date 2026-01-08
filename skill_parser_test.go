package goskills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillPackage(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a dummy SKILL.md file
	skillContent := `---
name: Test Skill
description: A skill for testing purposes.
allowed-tools: ["tool1", "tool2"]
model: gpt-4
author: Gemini
version: 0.1.0
license: MIT
---
# Test Skill Title

This is the main body of the skill. It contains instructions and other markdown content.

## Section 1
- Item 1
- Item 2

` + "```bash" + `
echo "Hello from bash"
` + "```" + `

## Section 2
More content here.
`
	skillPath := filepath.Join(tmpDir, "test-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillContent), 0644)
	assert.NoError(t, err)

	// Create dummy resource files
	err = os.Mkdir(filepath.Join(skillPath, "scripts"), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(skillPath, "scripts", "test.sh"), []byte("echo 'hello'"), 0644)
	assert.NoError(t, err)

	err = os.Mkdir(filepath.Join(skillPath, "references"), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(skillPath, "references", "doc.txt"), []byte("some reference"), 0644)
	assert.NoError(t, err)

	err = os.Mkdir(filepath.Join(skillPath, "assets"), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(skillPath, "assets", "image.png"), []byte("image data"), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.NoError(t, err)
	assert.NotNil(t, pkg)

	assert.Equal(t, skillPath, pkg.Path)
	assert.Equal(t, "Test Skill", pkg.Meta.Name)
	assert.Equal(t, "A skill for testing purposes.", pkg.Meta.Description)
	assert.Equal(t, []string{"tool1", "tool2"}, pkg.Meta.AllowedTools)
	assert.Equal(t, "gpt-4", pkg.Meta.Model)
	assert.Equal(t, "Gemini", pkg.Meta.Author)
	assert.Equal(t, "0.1.0", pkg.Meta.Version)
	assert.Equal(t, "MIT", pkg.Meta.License)

	// Check the raw body content
	expectedBody := `# Test Skill Title

This is the main body of the skill. It contains instructions and other markdown content.

## Section 1
- Item 1
- Item 2

` + "```bash" + `
echo "Hello from bash"
` + "```" + `

## Section 2
More content here.`
	assert.Equal(t, strings.TrimSpace(expectedBody), strings.TrimSpace(pkg.Body))

	assert.Len(t, pkg.Resources.Scripts, 1)
	assert.Equal(t, filepath.Join("scripts", "test.sh"), pkg.Resources.Scripts[0])

	assert.Len(t, pkg.Resources.References, 1)
	assert.Equal(t, filepath.Join("references", "doc.txt"), pkg.Resources.References[0])

	assert.Len(t, pkg.Resources.Assets, 1)
	assert.Equal(t, filepath.Join("assets", "image.png"), pkg.Resources.Assets[0])
}

func TestParseSkillPackage_NoFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "no-frontmatter-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("Just some markdown content."), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.Error(t, err)
	assert.Nil(t, pkg)
	assert.Contains(t, err.Error(), "no YAML frontmatter found")
}

func TestParseSkillPackage_InvalidFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "invalid-frontmatter-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	invalidContent := `---
name: Test Skill
description: A skill for testing purposes.
allowed-tools: ["tool1", "tool2"]
invalid-key: [
---
# Body
`
	err = os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(invalidContent), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.Error(t, err)
	assert.Nil(t, pkg)
	assert.Contains(t, err.Error(), "failed to parse SKILL.md frontmatter")
}

func TestParseSkillPackage_NoSkillMD(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "empty-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.Error(t, err)
	assert.Nil(t, pkg)
	assert.Contains(t, err.Error(), "neither SKILL.md nor skill.md found")
}

func TestParseSkillPackage_NonExistentDir(t *testing.T) {
	pkg, err := ParseSkillPackage("/non/existent/path")
	assert.Error(t, err)
	assert.Nil(t, pkg)
	assert.Contains(t, err.Error(), "skill directory not found")
}

func TestParseSkillPackage_EmptyResources(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "empty-resources-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	skillContent := `---
name: Empty Resources Skill
description: A skill with no resources.
allowed-tools: []
---
# Body
`
	err = os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillContent), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.NoError(t, err)
	assert.NotNil(t, pkg)

	assert.Empty(t, pkg.Resources.Scripts)
	assert.Empty(t, pkg.Resources.References)
	assert.Empty(t, pkg.Resources.Assets)
}

func TestParseSkillPackage_SubdirectoriesInResources(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "sub-resources-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	skillContent := `---
name: Subdir Resources Skill
description: A skill with resources in subdirectories.
allowed-tools: []
---
# Body
`
	err = os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(skillContent), 0644)
	assert.NoError(t, err)

	// Create nested resource files
	err = os.MkdirAll(filepath.Join(skillPath, "scripts", "subdir"), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(skillPath, "scripts", "subdir", "nested.sh"), []byte("echo 'nested'"), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.NoError(t, err)
	assert.NotNil(t, pkg)

	assert.Len(t, pkg.Resources.Scripts, 1)

	assert.Equal(t, filepath.Join("scripts", "subdir", "nested.sh"), pkg.Resources.Scripts[0])
}

func TestParseSkillPackages(t *testing.T) {
	skills, err := ParseSkillPackages("./testdata/skills")
	require.NoError(t, err)
	// Allow for flexible number of skills as new skills may be added
	require.Greater(t, len(skills), 25)
}

func TestParseOpenAISkillPackage(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a dummy skill.md file (OpenAI format)
	skillContent := `# Spreadsheet Skill (Create • Edit • Analyze • Visualize)

Use this skill when you need to work with spreadsheets (.xlsx, .csv, .tsv) to do any of the following:
- Create a new workbook/sheet with proper formulas, cell/number formatting, and structured layout
- Read or analyze tabular data (filter, aggregate, pivot, compute metrics) directly in a sheet
- Modify an existing workbook without breaking existing formulas or references
- Visualize data with in-sheet charts/tables and sensible formatting
- Recalculate/evaluate formulas to update results after changes

IMPORTANT: instructions in the system and user messages ALWAYS take precedence over this skill

## Guidelines for working with spreadsheets

### Use the artifact_tool python library or openpyxl
- You can use either python library (openpyxl or artifact_tool) for creating and editing spreadsheets
`
	skillPath := filepath.Join(tmpDir, "spreadsheet-skill")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(skillPath, "skill.md"), []byte(skillContent), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.NoError(t, err)
	assert.NotNil(t, pkg)

	assert.Equal(t, skillPath, pkg.Path)
	// Name should come from directory name
	assert.Equal(t, "spreadsheet skill", pkg.Meta.Name)
	// Description should be extracted from between first # and first ##
	assert.Contains(t, pkg.Meta.Description, "Use this skill when you need to work with spreadsheets")
	assert.Contains(t, pkg.Meta.Description, "Create a new workbook/sheet")

	// Check that allowed tools were inferred
	assert.NotEmpty(t, pkg.Meta.AllowedTools)
	assert.Contains(t, pkg.Meta.AllowedTools, "read_file")
	assert.Contains(t, pkg.Meta.AllowedTools, "write_file")
	assert.Contains(t, pkg.Meta.AllowedTools, "run_python_code") // Should be inferred for spreadsheets

	// Check that environment mapping was added to body
	assert.Contains(t, pkg.Body, "工具使用")
	assert.Contains(t, pkg.Body, "基于你的历史经验")
	assert.Contains(t, pkg.Body, "Original Skill Content")
}

func TestParseOpenAISkillPackage_WithDashes(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a dummy skill.md file (OpenAI format)
	skillContent := `# DOCX reading, creation, and review guidance

## Reading DOCXs
- Use soffice to convert DOCXs to PDFs.
- Then convert PDF to page images for visual inspection.
`
	skillPath := filepath.Join(tmpDir, "docx-processor")
	err := os.Mkdir(skillPath, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(skillPath, "skill.md"), []byte(skillContent), 0644)
	assert.NoError(t, err)

	pkg, err := ParseSkillPackage(skillPath)
	assert.NoError(t, err)
	assert.NotNil(t, pkg)

	// Name should replace dashes with spaces
	assert.Equal(t, "docx processor", pkg.Meta.Name)
	// Description should be extracted (fallback since no paragraph between # and ##)
	assert.Equal(t, "docx processor", pkg.Meta.Description)
}

func TestParseOpenAISkillPackages(t *testing.T) {
	skills, err := ParseSkillPackages("./testdata/oai-skills")
	require.NoError(t, err)
	// Should find at least the 3 example OpenAI skills
	require.GreaterOrEqual(t, len(skills), 3)

	// Check that skills are properly parsed
	skillNames := make(map[string]bool)
	for _, skill := range skills {
		skillNames[skill.Meta.Name] = true
		// Verify that each skill has a name
		assert.NotEmpty(t, skill.Meta.Name)
		// Verify that each skill has a description
		assert.NotEmpty(t, skill.Meta.Description)
		// Verify that each skill has allowed tools inferred
		assert.NotEmpty(t, skill.Meta.AllowedTools)
		// Verify that environment mapping was added
		assert.Contains(t, skill.Body, "工具使用")
	}

	// Check for expected skill names
	assert.Contains(t, skillNames, "spreadsheets")
	assert.Contains(t, skillNames, "docs")
	assert.Contains(t, skillNames, "pdfs")
}
