package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goskills-cli",
	Short: "A CLI tool for creating and managing Claude skills.",
	Long: `goskills-cli is a command-line interface to help you develop, parse,
and manage Claude Skill packages.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Disable the default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}