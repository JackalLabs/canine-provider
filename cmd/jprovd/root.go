package cmd

import (
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
)

// NewRootCmd creates a new root command for wasmd. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	// encodingConfig := app.MakeEncodingConfig()

	// cfg := sdk.GetConfig()
	// cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	// cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	// cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	// cfg.Seal()

	// initClientCtx := client.Context{}.
	// 	WithCodec(encodingConfig.Marshaler).
	// 	WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
	// 	WithTxConfig(encodingConfig.TxConfig).
	// 	WithLegacyAmino(encodingConfig.Amino).
	// 	WithInput(os.Stdin).
	// 	WithAccountRetriever(authtypes.AccountRetriever{}).
	// 	WithBroadcastMode(flags.BroadcastBlock).
	// 	WithHomeDir(app.DefaultNodeHome).
	// 	WithViper("")

	rootCmd := &cobra.Command{
		Use:   version.AppName,
		Short: "Provider Daemon (server)",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			// initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			// if err != nil {
			// 	return err
			// }

			// initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			// if err != nil {
			// 	return err
			// }

			// if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
			// 	return err
			// }

			return server.InterceptConfigsPreRunHandler(cmd, "", nil)
		},
	}

	rootCmd.AddCommand(StartServer())

	return rootCmd
}
