package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "pippaothy",
	Short: "Pippaothy - A personal web application",
	Long: `Pippaothy is a personal web application built with Go.
It provides both a web server and other utilities for managing the application`,
}

// Execute runs the root command
func Execute() {
	err := root.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add commands
	root.AddCommand(serveCmd)
	// Add more commands here as your application grows
	// root.AddCommand(migrateCmd)
	// root.AddCommand(seedCmd)
	// root.AddCommand(versionCmd)
}