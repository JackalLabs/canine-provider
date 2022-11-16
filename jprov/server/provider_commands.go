package server

import (
	"github.com/JackalLabs/jackal-provider/jprov/types"

	"github.com/spf13/cobra"
)

func StartServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start jackal storage provider",
		Long:  `Start jackal storage provider`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			StartFileServer(cmd)
			return nil
		},
	}

	// flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(types.DataDir, types.DefaultAppHome, "Data folder for the Jackal Storage Provider.")
	cmd.Flags().String("port", "3333", "Port to host the server on.")
	cmd.Flags().Bool("debug", false, "Allow the printing of info messages from the Storage Provider.")
	cmd.Flags().Uint16("interval", 30, "The interval in seconds for which to check proofs.")

	return cmd
}
