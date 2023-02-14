package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/JackalLabs/jackal-provider/jprov/server"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/spf13/cobra"
	tmcli "github.com/tendermint/tendermint/libs/cli"
)

func StartServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start jackal storage provider",
		Long:  `Start the Jackal Storage Provider server with the specified port.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			server.StartFileServer(cmd)
			return nil
		},
	}

	AddTxFlagsToCmd(cmd)
	cmd.Flags().String("port", "3333", "Port to host the server on.")
	cmd.Flags().Bool("debug", false, "Allow the printing of info messages from the Storage Provider.")
	cmd.Flags().String(types.VersionFlag, "", "The value exposed by the version api to allow for custom deployments.")
	cmd.Flags().Bool(types.HaltStraysFlag, false, "Debug flag to stop picking up strays.")
	cmd.Flags().Uint16(types.FlagInterval, 10, "The interval in seconds for which to check proofs.")
	cmd.Flags().Uint(types.FlagThreads, 10, "The amount of stray threads.")
	cmd.Flags().Int(types.FlagMaxMisses, 16, "The amount of intervals a provider can miss their proofs before removing a file.")
	cmd.Flags().Int64(types.FlagChunkSize, 1024*20, "The size of a single chunk.")

	return cmd
}

func DataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Provider data commands",
		Long:  `The sub-menu for Jackal Storage Provider data commands.`,
	}

	cmds := []*cobra.Command{
		CmdSetProviderTotalspace(),
		CmdSetProviderIP(),
		CmdSetProviderKeybase(),
	}

	for _, c := range cmds {
		AddTxFlagsToCmd(c)
		cmd.AddCommand(c)
	}

	return cmd
}

func NetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Provider network commands",
		Long:  `The sub-menu for Jackal Storage Provider network commands.`,
	}

	cmd.AddCommand(
		rpc.StatusCommand(),
		GetIpCmd(),
	)

	return cmd
}

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints version info",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%s %s\n", version.AppName, version.Version)
			return nil
		},
	}

	return cmd
}

func GetIpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Get provider ip address",
		Long:  `Get the external ip address this provider can be connected to.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := server.PickRouterClient(cmd.Context())
			if err != nil {
				return err
			}

			externalIP, err := cli.GetExternalIPAddress()
			if err != nil {
				return err
			}
			fmt.Printf("%s:3333\n", externalIP)

			return nil
		},
	}

	return cmd
}

func ResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset storage provider",
		Long:  `Resets the storage provider, this includes removing the storage directory & the internal database but keeping the private key.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			buf := bufio.NewReader(cmd.InOrStdin())

			yes, err := input.GetConfirmation("Are you sure you want to reset the system?", buf, cmd.ErrOrStderr())
			if err != nil {
				return err
			}

			if !yes {
				return nil
			}

			err = os.RemoveAll(utils.GetStorageAllPath(clientCtx))
			if err != nil {
				return err
			}

			err = os.RemoveAll(utils.GetDataPath(clientCtx))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// AddTxFlagsToCmd adds common flags to a module tx command.
func AddTxFlagsToCmd(cmd *cobra.Command) {
	cmd.Flags().StringP(tmcli.OutputFlag, "o", "json", "Output format (text|json)")
	cmd.Flags().Uint64P(flags.FlagAccountNumber, "a", 0, "The account number of the signing account (offline mode only)")
	cmd.Flags().Uint64P(flags.FlagSequence, "s", 0, "The sequence number of the signing account (offline mode only)")
	cmd.Flags().String(flags.FlagNote, "", "Note to add a description to the transaction (previously --memo)")
	cmd.Flags().String(flags.FlagFees, "", "Fees to pay along with transaction; eg: 10ujkl")
	cmd.Flags().String(flags.FlagGasPrices, "0.02ujkl", "Gas prices in decimal format to determine the transaction fee (e.g. 0.1ujkl)")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
	cmd.Flags().Bool(flags.FlagUseLedger, false, "Use a connected Ledger device")
	cmd.Flags().Float64(flags.FlagGasAdjustment, flags.DefaultGasAdjustment, "adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored ")
	cmd.Flags().StringP(flags.FlagBroadcastMode, "b", flags.BroadcastBlock, "Transaction broadcasting mode (sync|async|block)")
	cmd.Flags().Bool(flags.FlagDryRun, false, "ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it (when enabled, the local Keybase is not accessible)")
	cmd.Flags().Bool(flags.FlagGenerateOnly, false, "Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase is not accessible)")
	cmd.Flags().Bool(flags.FlagOffline, false, "Offline mode (does not allow any online functionality")
	cmd.Flags().BoolP(flags.FlagSkipConfirmation, "y", false, "Skip tx broadcasting prompt confirmation")
	cmd.Flags().String(flags.FlagSignMode, "", "Choose sign mode (direct|amino-json), this is an advanced feature")
	cmd.Flags().Uint64(flags.FlagTimeoutHeight, 0, "Set a block timeout height to prevent the tx from being committed past a certain height")
	cmd.Flags().String(flags.FlagFeeAccount, "", "Fee account pays fees for the transaction instead of deducting from the signer")
	cmd.Flags().String(flags.FlagChainID, "", "The network chain ID")

	// --gas can accept integers and "auto"
	cmd.Flags().String(flags.FlagGas, "", fmt.Sprintf("gas limit to set per-transaction; set to %q to calculate sufficient gas automatically (default %d)", flags.GasFlagAuto, flags.DefaultGasLimit))
}
