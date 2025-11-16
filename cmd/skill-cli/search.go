package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smallnest/goskills"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [path] [query]",
	Short: "Searches for skills by name or description.",
	Long: `The search command scans a directory for valid skill packages and returns a list
	of skills where the name or description contains the provided query text.
	The search is case-insensitive.`, 
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillsRoot := args[0]
		query := strings.ToLower(args[1])

		entries, err := os.ReadDir(skillsRoot)
		if err != nil {
			return fmt.Errorf("could not read skills directory '%s': %w", skillsRoot, err)
		}

		fmt.Printf("--- Searching for '%s' in %s ---\n", query, skillsRoot)
		foundCount := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillPath := filepath.Join(skillsRoot, entry.Name())
			skillPackage, err := goskills.ParseSkillPackage(skillPath)
			if err != nil {
				continue // Not a valid skill
			}

			// Case-insensitive search in name and description
			name := strings.ToLower(skillPackage.Meta.Name)
			description := strings.ToLower(skillPackage.Meta.Description)

			if strings.Contains(name, query) || strings.Contains(description, query) {
				fmt.Printf("- %-20s: %s\n", skillPackage.Meta.Name, skillPackage.Meta.Description)
				foundCount++
			}
		}

		if foundCount == 0 {
			fmt.Println("No matching skills found.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
