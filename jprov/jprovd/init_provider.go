package main

import (
	"fmt"
	"strconv"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdInitProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [ip] [totalspace] [keybase-identity]",
		Short: "Init provider",
		Long:  "Initialize a provider with given parameters.",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argIP := args[0]
			argTotalspace := args[1]
			argKeybase := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				return err
			}
			fmt.Printf("Initializing account: %s\n", address)
			msg := types.NewMsgInitProvider(
				address,
				argIP,
				argTotalspace,
				argKeybase,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			_, err = utils.SendTx(clientCtx, cmd.Flags(), msg)
			return err
		},
	}

	return cmd
}
