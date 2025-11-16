package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/smallnest/goskills"
	"github.com/spf13/cobra"
)

var parseCmd = &cobra.Command{
	Use:   "parse <skill_directory>",
	Short: "Parses a skill directory and prints its metadata and a snippet of its body.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillDir := args[0]
		absSkillDir, err := filepath.Abs(skillDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", skillDir, err)
		}

		skillPackage, err := goskills.ParseSkillPackage(absSkillDir)
		if err != nil {
			return fmt.Errorf("failed to parse skill package: %w", err)
		}

		fmt.Printf("Skill Name: %s\n", skillPackage.Meta.Name)
		fmt.Printf("Description: %s\n", skillPackage.Meta.Description)
		fmt.Printf("Allowed Tools: %s\n", strings.Join(skillPackage.Meta.AllowedTools, ", "))
		if skillPackage.Meta.Model != "" {
			fmt.Printf("Model: %s\n", skillPackage.Meta.Model)
		}
		if skillPackage.Meta.Author != "" {
			fmt.Printf("Author: %s\n", skillPackage.Meta.Author)
		}
		if skillPackage.Meta.Version != "" {
			fmt.Printf("Version: %s\n", skillPackage.Meta.Version)
		}
		if skillPackage.Meta.License != "" {
			fmt.Printf("License: %s\n", skillPackage.Meta.License)
		}

		fmt.Println("\n--- Body Snippet (first 500 chars) ---")
		if len(skillPackage.Body) > 500 {
			fmt.Println(skillPackage.Body[:500] + "...")
		} else {
			fmt.Println(skillPackage.Body)
		}

		fmt.Println("\n--- Resources ---")
		if len(skillPackage.Resources.Scripts) > 0 {
			fmt.Println("Scripts:")
			for _, s := range skillPackage.Resources.Scripts {
				fmt.Printf("  - %s\n", s)
			}
		}
		if len(skillPackage.Resources.References) > 0 {
			fmt.Println("References:")
			for _, r := range skillPackage.Resources.References {
				fmt.Printf("  - %s\n", r)
			}
		}
		if len(skillPackage.Resources.Assets) > 0 {
			fmt.Println("Assets:")
			for _, a := range skillPackage.Resources.Assets {
				fmt.Printf("  - %s\n", a)
			}
		}
		if len(skillPackage.Resources.Scripts) == 0 && len(skillPackage.Resources.References) == 0 && len(skillPackage.Resources.Assets) == 0 {
			fmt.Println("No resources found.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}