package goskills

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// SkillMeta corresponds to the content of SKILL.md frontmatter
type SkillMeta struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
	Model        string   `yaml:"model,omitempty"`
	Author       string   `yaml:"author,omitempty"`
	Version      string   `yaml:"version,omitempty"`
	License      string   `yaml:"license,omitempty"`
}

// SkillResources lists the relevant resource files in the skill package
type SkillResources struct {
	Scripts    []string `json:"scripts"`
	References []string `json:"references"`
	Assets     []string `json:"assets"`
}

// extractFrontmatterAndBody separates and parses the frontmatter and body of SKILL.md
func extractFrontmatterAndBody(data []byte) (SkillMeta, string, error) {
	marker := []byte("---")
	var meta SkillMeta
	var body string

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

	// 1. Parse SKILL.md
	skillMdPath := filepath.Join(dirPath, "SKILL.md")
	mdContent, err := os.ReadFile(skillMdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SKILL.md not found in skill directory: %s", dirPath)
		}
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	meta, bodyStr, err := extractFrontmatterAndBody(mdContent)
	if err != nil {
		return nil, err
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

	// 3. Assemble SkillPackage
	pkg := &SkillPackage{
		Path: dirPath,
		Meta: meta,
		Body: bodyStr, // Store raw markdown body
		Resources: SkillResources{
			Scripts:    scripts,
			References: references,
			Assets:     assets,
		},
	}

	return pkg, nil
}