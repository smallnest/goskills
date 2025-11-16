package main

import (
	"fmt"
	"strings"

	"github.com/smallnest/goskills"
	"github.com/spf13/cobra"
)

var detailCmd = &cobra.Command{
	Use:   "detail [path]",
	Short: "Displays detailed information about a skill.",
	Long: `The detail command provides a comprehensive view of a skill package,
including all metadata and the full, unabridged content of the SKILL.md body.`, 
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillPath := args[0]
		skillPackage, err := goskills.ParseSkillPackage(skillPath)
		if err != nil {
			return fmt.Errorf("failed to parse skill: %w", err)
		}

		// --- Print Meta ---
		fmt.Printf("--- Skill: %s ---\n\n", skillPackage.Meta.Name)
		fmt.Println("[Meta]")
		fmt.Printf("  Description: %s\n", skillPackage.Meta.Description)
		if skillPackage.Meta.Model != "" {
			fmt.Printf("  Model: %s\n", skillPackage.Meta.Model)
		}
		if len(skillPackage.Meta.AllowedTools) > 0 {
			fmt.Printf("  Allowed Tools: %v\n", skillPackage.Meta.AllowedTools)
		}
		if skillPackage.Meta.Author != "" {
			fmt.Printf("  Author: %s\n", skillPackage.Meta.Author)
		}
		if skillPackage.Meta.Version != "" {
			fmt.Printf("  Version: %s\n", skillPackage.Meta.Version)
		}
		fmt.Println()

		// --- Print Resources ---
		fmt.Println("[Resources]")
		if len(skillPackage.Resources.Scripts) == 0 && len(skillPackage.Resources.References) == 0 && len(skillPackage.Resources.Assets) == 0 {
			fmt.Println("  No resource files found.")
		} else {
			if len(skillPackage.Resources.Scripts) > 0 {
				fmt.Printf("  Scripts: %v\n", skillPackage.Resources.Scripts)
			}
			if len(skillPackage.Resources.References) > 0 {
				fmt.Printf("  References: %v\n", skillPackage.Resources.References)
			}
			if len(skillPackage.Resources.Assets) > 0 {
				fmt.Printf("  Assets: %v\n", skillPackage.Resources.Assets)
			}
		}
		fmt.Println()

		// --- Print Full Body ---
		fmt.Println("[Full Body]")
		for _, part := range skillPackage.Body {
			switch p := part.(type) {
			case goskills.TitlePart:
				fmt.Printf("\n[Title]: %s\n", p.Text)
			case goskills.SectionPart:
				fmt.Printf("\n[Section]: %s\n", p.Title)
				fmt.Println(strings.Repeat("-", len(p.Title)+12))
				fmt.Println(p.Content)
			case goskills.MarkdownPart:
				fmt.Println(p.Content)
			case goskills.ImplementationPart:
				fmt.Printf("\n[Implementation]: %s\n", p.Language)
				fmt.Println("```" + p.Language)
				fmt.Print(p.Code)
				fmt.Println("```")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(detailCmd)
}
