package main

import (
	"github.com/nuwa/bpp.v3/cmd"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "app"}
	for _, it := range cmd.Command() {
		rootCmd.AddCommand(it)
	}
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		return
	}
}
