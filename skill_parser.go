package goskills

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillBodyPart is an interface for different parts of the skill's markdown body.
type SkillBodyPart interface {
	PartType() string
}

// TitlePart represents a [Title]: ... part.
type TitlePart struct {
	Text string
}
func (p TitlePart) PartType() string { return "Title" }

// SectionPart represents a [Section]: title: ... part.
type SectionPart struct {
	Title   string
	Content string
}
func (p SectionPart) PartType() string { return "Section" }

// MarkdownPart represents a block of plain markdown text.
type MarkdownPart struct {
	Content string
}
func (p MarkdownPart) PartType() string { return "Markdown" }

// ImplementationPart represents an implementation block.
type ImplementationPart struct {
	Language string
	Code     string
}
func (p ImplementationPart) PartType() string { return "Implementation" }


// SkillPackage represents a fully and finely parsed Claude Skill package
type SkillPackage struct {
	Path      string          `json:"path"`
	Meta      SkillMeta       `json:"meta"`
	Body      []SkillBodyPart `json:"body"` // Structured content of SKILL.md
	Resources SkillResources  `json:"resources"`
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

var (
	titleRegex = regexp.MustCompile(`^[Title]\s*:\s*(.*)`)
	sectionRegex = regexp.MustCompile(`^[Section]\s*:\s*title:\s*"(.*)"`) // Corrected regex for section title
	implRegex = regexp.MustCompile(`^This is the implementation in (.*)`)
)

// parseMarkdownBody parses the raw markdown string into structured parts.
func parseMarkdownBody(body string) []SkillBodyPart {
	var parts []SkillBodyPart
	scanner := bufio.NewScanner(strings.NewReader(body))
	
	var currentContent strings.Builder
	inCodeBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
		}

		if !inCodeBlock {
			if match := titleRegex.FindStringSubmatch(line); len(match) > 1 {
				if currentContent.Len() > 0 {
					parts = append(parts, MarkdownPart{Content: strings.TrimSpace(currentContent.String())})
					currentContent.Reset()
				}
				parts = append(parts, TitlePart{Text: match[1]})
				continue
			}
			if match := sectionRegex.FindStringSubmatch(line); len(match) > 1 {
				if currentContent.Len() > 0 {
					parts = append(parts, MarkdownPart{Content: strings.TrimSpace(currentContent.String())})
					currentContent.Reset()
				}
				parts = append(parts, SectionPart{Title: match[1]}) // Content will be the next markdown block
				continue
			}
			if match := implRegex.FindStringSubmatch(line); len(match) > 1 {
				if currentContent.Len() > 0 {
					// The content before this line is the content of the previous section
					if lastPart, ok := parts[len(parts)-1].(SectionPart); ok {
						lastPart.Content = strings.TrimSpace(currentContent.String())
						parts[len(parts)-1] = lastPart
					} else {
						parts = append(parts, MarkdownPart{Content: strings.TrimSpace(currentContent.String())})
					}
					currentContent.Reset()
				}
				// The code block follows this line
				var codeContent strings.Builder
				scanner.Scan() // Move to ``` line
				for scanner.Scan() {
					codeLine := scanner.Text()
					if strings.HasPrefix(codeLine, "```") {
						break
					}
					codeContent.WriteString(codeLine + "\n")
				}
				parts = append(parts, ImplementationPart{Language: match[1], Code: codeContent.String()})
				continue
			}
		}
		currentContent.WriteString(line + "\n")
	}

	if currentContent.Len() > 0 {
		// Assign remaining content to the last section or as a general markdown part
		if len(parts) > 0 {
			if lastPart, ok := parts[len(parts)-1].(SectionPart); ok && lastPart.Content == "" {
				lastPart.Content = strings.TrimSpace(currentContent.String())
				parts[len(parts)-1] = lastPart
			} else {
				parts = append(parts, MarkdownPart{Content: strings.TrimSpace(currentContent.String())})
			}
		} else {
			parts = append(parts, MarkdownPart{Content: strings.TrimSpace(currentContent.String())})
		}
	}

	return parts
}


// extractAndParseSKILLmd separates and parses the frontmatter and body of SKILL.md
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

	bodyParts := parseMarkdownBody(bodyStr)

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
		Body: bodyParts,
		Resources: SkillResources{
			Scripts:    scripts,
			References: references,
			Assets:     assets,
		},
	}

	return pkg, nil
}
