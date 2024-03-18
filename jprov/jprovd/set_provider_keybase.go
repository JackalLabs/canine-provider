package main

import (
	"strconv"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetProviderKeybase() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-key [keybase-identity]",
		Short: "Set providers keybase identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argKey := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetProviderKeybase(
				address,
				argKey,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			_, err = utils.SendTx(clientCtx, cmd.Flags(), "", msg)
			return err
		},
	}

	return cmd
}
