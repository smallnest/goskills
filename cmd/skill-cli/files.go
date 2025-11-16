package main

import (
	"fmt"
	"path/filepath"

	"github.com/smallnest/goskills"
	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files [path]",
	Short: "Lists all files comprising a skill package.",
	Long: `The files command parses a skill package and lists all the files that make it up,
including the SKILL.md file and all discovered resource files.`, 
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillPath := args[0]
		skillPackage, err := goskills.ParseSkillPackage(skillPath)
		if err != nil {
			return fmt.Errorf("failed to parse skill: %w", err)
		}

		fmt.Printf("Files for skill: %s\n", skillPackage.Meta.Name)
		
		// Add the SKILL.md file itself
		fmt.Printf("- %s\n", filepath.Join(skillPackage.Path, "SKILL.md"))

		// Add all resource files
		resources := skillPackage.Resources
		for _, file := range resources.Scripts {
			fmt.Printf("- %s\n", filepath.Join(skillPackage.Path, file))
		}
		for _, file := range resources.References {
			fmt.Printf("- %s\n", filepath.Join(skillPackage.Path, file))
		}
		for _, file := range resources.Assets {
			fmt.Printf("- %s\n", filepath.Join(skillPackage.Path, file))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
}
