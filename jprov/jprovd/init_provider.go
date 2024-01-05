package main

import (
	"fmt"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/spf13/cobra"
)

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
				fmt.Println(err)
				return err
			}

			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				fmt.Println(err)
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
				fmt.Println(err)
				return err
			}
			res, err := utils.SendTx(clientCtx, cmd.Flags(), "", msg)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println(res.RawLog)
			return err
		},
	}

	return cmd
}
