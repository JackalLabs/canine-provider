package main

import (
	"fmt"
	"os"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/jackalLabs/canine-chain/app"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	encodingConfig := app.MakeEncodingConfig()

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	cfg.Seal()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastBlock).
		WithHomeDir(types.DefaultAppHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   "jprovd",
		Short: "Provider Daemon (server)",
		Long:  "Jackal Lab's implimentation of a Jackal Protocol Storage Provider system.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				fmt.Println(err)
				return err
			}

			initClientCtx, err = ReadFromClientConfig(initClientCtx)
			if err != nil {
				fmt.Println(err)
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				fmt.Println(err)
				return err
			}

			return utils.InterceptConfigsPreRunHandler(cmd, "", nil)
		},
	}

	init := CmdInitProvider()
	AddTxFlagsToCmd(init)
	init.Flags().Int(types.FlagGasCap, 3_000_000, "The maximum gas to be used per message.")

	rootCmd.AddCommand(
		StartServerCommand(),
		ResetCommand(),
		init,
		DataCmd(),
		ClientCmd(),
		VersionCmd(),
		NetworkCmd(),
		BlanketCmd(),
	)

	return rootCmd
}
