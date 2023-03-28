package main

import (
	"strconv"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetProviderIP() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-ip [ip]",
		Short: "Set provider's ip address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argIP := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetProviderIP(
				address,
				argIP,
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
