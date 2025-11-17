package main

import (
	"fmt"

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

		packages, err := goskills.ParseSkillPackages(skillsRoot)
		if err != nil {
			return fmt.Errorf("could not parse skills in directory '%s': %w", skillsRoot, err)
		}

		fmt.Printf("--- Skills found in %s ---\n", skillsRoot)
		if len(packages) == 0 {
			fmt.Println("No valid skills found.")
			return nil
		}

		for _, skillPackage := range packages {
			fmt.Printf("- %-20s: %s\n", skillPackage.Meta.Name, skillPackage.Meta.Description)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
