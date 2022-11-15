package jprovd

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "jprovd",
		Short: "Provider Daemon (server)",
		Long:  "Jackal Lab's implimentation of a Jackal Protocol Storage Provider system.",
	}

	rootCmd.AddCommand(StartServer())

	return rootCmd
}
