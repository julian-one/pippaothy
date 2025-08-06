package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pippaothy",
	Short: "Pippaothy - A personal web application",
	Long: `Pippaothy is a personal web application built with Go.
It provides both a web server and other utilities for managing the application`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(scraperCmd)
}
