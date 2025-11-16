package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smallnest/goskills"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "Lists all valid skills in a given directory.",
	Long: `The list command scans a directory for subdirectories that are valid 
Claude skill packages and prints a summary of each one found.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillsRoot := args[0]
		
		entries, err := os.ReadDir(skillsRoot)
		if err != nil {
			return fmt.Errorf("could not read skills directory '%s': %w", skillsRoot, err)
		}

		fmt.Printf("--- Skills found in %s ---\n", skillsRoot)
		parsedCount := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillPath := filepath.Join(skillsRoot, entry.Name())
			skillPackage, err := goskills.ParseSkillPackage(skillPath)
			if err != nil {
				// Not a valid skill, just skip it.
				continue
			}

			fmt.Printf("- %-20s: %s\n", skillPackage.Meta.Name, skillPackage.Meta.Description)
			parsedCount++
		}

		if parsedCount == 0 {
			fmt.Println("No valid skills found.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
