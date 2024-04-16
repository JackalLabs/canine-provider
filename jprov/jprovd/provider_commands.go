package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/JackalLabs/blanket/blanket"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/server"
	"github.com/JackalLabs/jackal-provider/jprov/strays"
	"github.com/JackalLabs/jackal-provider/jprov/types"
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
			clientCtx := client.GetClientContextFromCmd(cmd)
			dbPath := utils.GetArchiveDBPath(clientCtx)
			archivedb, err := archive.NewDoubleRefArchiveDB(dbPath)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, archivedb.Close())
			}()

			downtimedbPath := utils.GetDowntimeDBPath(clientCtx)
			downtimedb, err := archive.NewDowntimeDB(downtimedbPath)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, downtimedb.Close())
			}()

			// start stray service
			if haltStray, err := cmd.Flags().GetBool(types.HaltStraysFlag); err != nil {
				return err
			} else if !haltStray {
				manager, err := strays.NewStrayManager(cmd, archivedb, downtimedb)
				if err != nil {
					return err
				}

				err = manager.Init()
				if err != nil {
					return err
				}
				go manager.Start()
			}

			fs, err := server.NewFileServer(cmd, archivedb, downtimedb)
			if err != nil {
				return err
			}
			fs.StartFileServer(cmd)
			return nil
		},
	}

	AddTxFlagsToCmd(cmd)
	cmd.Flags().Int(types.FlagPort, types.DefaultPort, "Port to host the server on.")
	cmd.Flags().String(types.VersionFlag, "", "The value exposed by the version api to allow for custom deployments.")
	cmd.Flags().Bool(types.HaltStraysFlag, false, "Debug flag to stop picking up strays.")
	cmd.Flags().Uint16(types.FlagInterval, types.DefaultInterval, "The interval in seconds for which to check proofs. Must be >=1800 if you need a custom interval")
	cmd.Flags().Uint(types.FlagThreads, types.DefaultThreads, "The amount of stray threads.")
	cmd.Flags().Int(types.FlagMaxMisses, types.DefaultMaxMisses, "The amount of intervals a provider can miss their proofs before removing a file.")
	cmd.Flags().Int64(types.FlagChunkSize, types.DefaultChunkSize, "The size of a single file chunk.")
	cmd.Flags().Int64(types.FlagStrayInterval, types.DefaultStrayInterval, "The interval in seconds to check for new strays.")
	cmd.Flags().Int(types.FlagMessageSize, types.DefaultMessageSize, "The max size of all messages in bytes to submit to the chain at one time.")
	cmd.Flags().Int(types.FlagGasCap, types.DefaultGasCap, "The maximum gas to be used per message.")
	cmd.Flags().Int(types.FlagMaxFileSize, types.DefaultMaxMisses, "The maximum size allowed to be sent to this provider in mbs. (only for monitoring services)")
	cmd.Flags().Int64(types.FlagQueueInterval, types.DefaultQueueInterval, "The time, in seconds, between running a queue loop.")
	cmd.Flags().String(types.FlagProviderName, "A Storage Provider", "The name to identify this provider in block explorers.")
	cmd.Flags().Int64(types.FlagSleep, types.DefaultSleep, "The time, in milliseconds, before adding another proof msg to the queue.")
	cmd.Flags().Bool(types.FlagDoReport, types.DefaultDoReport, "Should this provider report deals (uses gas).")
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
		CmdDumpDatabase(),
	}

	for _, c := range cmds {
		AddTxFlagsToCmd(c)
		cmd.AddCommand(c)
	}

	return cmd
}

func BlanketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blanket",
		Short: "Monitor Jackal Provider Daemon with a terminal GUI",
		Long:  "Monitor Jackal Provider Daemon with Blanket, the terminal GUI for inspecting critical functions of Jackal Providers & the Jackal Protocol Network",
		RunE: func(cmd *cobra.Command, args []string) error {
			blanket.CmdRunBlanket("http://127.0.0.1:3333")
			return nil
		},
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

			err = os.RemoveAll(utils.GetArchiveDBPath(clientCtx))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func PruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune files that are no longer on contract according to chain data",
		Long:  "Prune files that are no longer on contract according to chain data",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			buf := bufio.NewReader(cmd.InOrStdin())
			yes, err := input.GetConfirmation("Are you sure you want to prune expired files?", buf, cmd.ErrOrStderr())
			if err != nil {
				return err
			}

			if !yes {
				return nil
			}

			clientCtx := client.GetClientContextFromCmd(cmd)

			dbPath := utils.GetArchiveDBPath(clientCtx)
			archivedb, err := archive.NewDoubleRefArchiveDB(dbPath)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, archivedb.Close())
			}()

			downtimedbPath := utils.GetDowntimeDBPath(clientCtx)
			downtimedb, err := archive.NewDowntimeDB(downtimedbPath)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, downtimedb.Close())
			}()

			fs, err := server.NewFileServer(cmd, archivedb, downtimedb)
			if err != nil {
				return err
			}

			err = fs.Init()
			if err != nil {
				return err
			}

			err = fs.RecollectActiveDeals()
			if err != nil {
				return err
			}

			interval, err := cmd.Flags().GetUint16(types.FlagInterval)
			if err != nil {
				interval = 0
			}

			fmt.Println("starting proof server")
			go fs.StartProofServer(interval)

			return fs.PruneExpiredFiles()
		},
	}

	cmd.Flags().Int64(types.FlagChunkSize, types.DefaultChunkSize, "The size of a single file chunk.")
	return cmd
}

func MigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate from old file system to new file system",
		Long:  `Migrate old file system. This will glue all blocks together into one file per fids stored in your machine`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			buf := bufio.NewReader(cmd.InOrStdin())
			yes, err := input.GetConfirmation("Are you sure you want to migrate from old file system?", buf, cmd.ErrOrStderr())
			if err != nil {
				return err
			}

			if !yes {
				return nil
			}

			defer func() {
				// required to stop proof server
				err = errors.Join(err, syscall.Kill(os.Getpid(), syscall.SIGTERM))
			}()

			clientCtx := client.GetClientContextFromCmd(cmd)
			dbPath := utils.GetArchiveDBPath(clientCtx)
			archivedb, err := archive.NewDoubleRefArchiveDB(dbPath)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, archivedb.Close())
			}()

			downtimedbPath := utils.GetDowntimeDBPath(clientCtx)
			downtimedb, err := archive.NewDowntimeDB(downtimedbPath)
			if err != nil {
				return err
			}
			defer func() {
				err = errors.Join(err, downtimedb.Close())
			}()

			fs, err := server.NewFileServer(cmd, archivedb, downtimedb)
			if err != nil {
				return err
			}

			err = fs.Init()
			if err != nil {
				return err
			}

			err = fs.RecollectActiveDeals()
			if err != nil {
				return err
			}

			interval, err := cmd.Flags().GetUint16(types.FlagInterval)
			if err != nil {
				interval = 0
			}

			fmt.Println("starting proof server")
			go fs.StartProofServer(interval)

			chunkSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
			if err != nil {
				err = errors.Join(errors.New("Migrate: cannot migrate without chunk size"), err)
				return err
			}

			utils.Migrate(clientCtx, chunkSize)
			return err
		},
	}
	AddTxFlagsToCmd(cmd)
	cmd.Flags().Int(types.FlagPort, types.DefaultPort, "Port to host the server on.")
	cmd.Flags().String(types.VersionFlag, "", "The value exposed by the version api to allow for custom deployments.")
	cmd.Flags().Bool(types.HaltStraysFlag, false, "Debug flag to stop picking up strays.")
	cmd.Flags().Uint16(types.FlagInterval, types.DefaultInterval, "The interval in seconds for which to check proofs. Must be >=1800 if you need a custom interval")
	cmd.Flags().Uint(types.FlagThreads, types.DefaultThreads, "The amount of stray threads.")
	cmd.Flags().Int(types.FlagMaxMisses, types.DefaultMaxMisses, "The amount of intervals a provider can miss their proofs before removing a file.")
	cmd.Flags().Int64(types.FlagChunkSize, types.DefaultChunkSize, "The size of a single file chunk.")
	cmd.Flags().Int64(types.FlagStrayInterval, types.DefaultStrayInterval, "The interval in seconds to check for new strays.")
	cmd.Flags().Int(types.FlagMessageSize, types.DefaultMessageSize, "The max size of all messages in bytes to submit to the chain at one time.")
	cmd.Flags().Int(types.FlagGasCap, types.DefaultGasCap, "The maximum gas to be used per message.")
	cmd.Flags().Int64(types.FlagQueueInterval, types.DefaultQueueInterval, "The time, in seconds, between running a queue loop.")
	cmd.Flags().String(types.FlagProviderName, "A Storage Provider", "The name to identify this provider in block explorers.")
	cmd.Flags().Int64(types.FlagSleep, types.DefaultSleep, "The time, in milliseconds, before adding another proof msg to the queue.")
	cmd.Flags().Bool(types.FlagDoReport, types.DefaultDoReport, "Should this provider report deals (uses gas).")

	return cmd
}

// AddTxFlagsToCmd adds common flags to a module tx command.
func AddTxFlagsToCmd(cmd *cobra.Command) {
	cmd.Flags().StringP(tmcli.OutputFlag, "o", "json", "Output format (text|json)")
	cmd.Flags().Uint64P(flags.FlagAccountNumber, "a", 0, "The account number of the signing account (offline mode only)")
	cmd.Flags().Uint64P(flags.FlagSequence, "s", 0, "The sequence number of the signing account (offline mode only)")
	cmd.Flags().String(flags.FlagNote, "", "Note to add a description to the transaction (previously --memo)")
	cmd.Flags().String(flags.FlagFees, "", "Fees to pay along with transaction; eg: 10ujkl")
	cmd.Flags().String(flags.FlagGasPrices, "0.002ujkl", "Gas prices in decimal format to determine the transaction fee (e.g. 0.1ujkl)")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
	cmd.Flags().Bool(flags.FlagUseLedger, false, "Use a connected Ledger device")
	cmd.Flags().Float64(flags.FlagGasAdjustment, 1.75, "adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored ")
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
	cmd.Flags().String(flags.FlagGas, "auto", fmt.Sprintf("gas limit to set per-transaction; set to %q to calculate sufficient gas automatically (default %d)", flags.GasFlagAuto, flags.DefaultGasLimit))
}
