package main

import (
	"encoding/json"
	"fmt"

	"github.com/smallnest/goskills"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var parseCmd = &cobra.Command{
	Use:   "parse [path]",
	Short: "Parses a Claude skill and displays its structure.",
	Long: `The parse command reads a skill directory, parses the SKILL.md file,
and prints a structured representation of its metadata, body, and resources.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillPath := args[0]
		skillPackage, err := goskills.ParseSkillPackage(skillPath)
		if err != nil {
			return fmt.Errorf("failed to parse skill: %w", err)
		}

		if jsonOutput {
			output, err := json.MarshalIndent(skillPackage, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format output as JSON: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		// Pretty print the output
		fmt.Printf("--- Skill: %s ---\n\n", skillPackage.Meta.Name)
		
		fmt.Println("[Meta]")
		fmt.Printf("  Description: %s\n", skillPackage.Meta.Description)
		if skillPackage.Meta.Model != "" {
			fmt.Printf("  Model: %s\n", skillPackage.Meta.Model)
		}
		if len(skillPackage.Meta.AllowedTools) > 0 {
			fmt.Printf("  Allowed Tools: %v\n", skillPackage.Meta.AllowedTools)
		}
		fmt.Println()

		fmt.Println("[Body]")
		for _, part := range skillPackage.Body {
			switch p := part.(type) {
			case goskills.TitlePart:
				fmt.Printf("  [Title]: %s\n", p.Text)
			case goskills.SectionPart:
				fmt.Printf("  [Section]: %s\n", p.Title)
			// fmt.Printf("    Content: %s\n", p.Content) // Can be verbose
			case goskills.MarkdownPart:
				fmt.Printf("  [Markdown]: (%.40s...)\n", p.Content)
			case goskills.ImplementationPart:
				fmt.Printf("  [Implementation]: %s\n", p.Language)
			}
		}
		fmt.Println()

		fmt.Println("[Resources]")
		if len(skillPackage.Resources.Scripts) > 0 {
			fmt.Printf("  Scripts: %v\n", skillPackage.Resources.Scripts)
		}
		if len(skillPackage.Resources.References) > 0 {
			fmt.Printf("  References: %v\n", skillPackage.Resources.References)
		}
		if len(skillPackage.Resources.Assets) > 0 {
			fmt.Printf("  Assets: %v\n", skillPackage.Resources.Assets)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
	parseCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output the parsed structure as JSON")
}
