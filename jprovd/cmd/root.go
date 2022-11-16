package main

import (
	"github.com/JackalLabs/jackal-provider/jprovd/server"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "jprovd",
		Short: "Provider Daemon (server)",
		Long:  "Jackal Lab's implimentation of a Jackal Protocol Storage Provider system.",
	}

	rootCmd.AddCommand(server.StartServer())

	return rootCmd
}
