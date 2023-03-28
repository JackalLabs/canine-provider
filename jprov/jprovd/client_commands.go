package main

import (
	"bufio"
	"context"
	"fmt"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	clientConfig "github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	stortypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

func ClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Provider client commands",
		Long:  `The sub-menu for Jackal Storage Provider client commands.`,
	}

	cmd.AddCommand(
		clientConfig.Cmd(),
		GenKeyCommand(),
		GetBalanceCmd(),
		GetAddressCmd(),
		WithdrawCommand(),
	)

	return cmd
}

func WithdrawCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw [address] [coins]",
		Short: "Withdraw tokens to a specified account.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				return err
			}

			fromAddr, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				return err
			}

			toAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinsNormalized(args[1])
			if err != nil {
				return err
			}

			msg := banktypes.NewMsgSend(fromAddr, toAddr, coins)

			res, err := utils.SendTx(clientCtx, cmd.Flags(), msg)
			if res != nil {
				fmt.Println(res.RawLog)
			}
			return err
		},
	}
	AddTxFlagsToCmd(cmd)

	return cmd
}

func GetAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address",
		Short: "Get account address",
		Long:  `Get the account address of the current storage provider key.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			fmt.Printf("Address: %s\n", address)

			return nil
		},
	}

	return cmd
}

func GetBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get account balance",
		Long:  `Get the account balance of the current storage provider key.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := banktypes.NewQueryClient(clientCtx)
			address, err := crypto.GetAddress(clientCtx)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			params := &banktypes.QueryBalanceRequest{
				Denom:   "ujkl",
				Address: address,
			}

			res, err := queryClient.Balance(context.Background(), params)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			fmt.Printf("Balance: %s\n", res.Balance)

			return nil
		},
	}

	return cmd
}

func GenKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-key",
		Short: "Generate a new private key",
		Long:  `Generate a new Jackal address and private key combination to interact with the blockchain.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			entropySeed, err := bip39.NewEntropy(256)
			if err != nil {
				return err
			}

			mnemonic, err := bip39.NewMnemonic(entropySeed)
			if err != nil {
				return err
			}

			pKey := secp256k1.GenPrivKeyFromSecret([]byte(mnemonic))

			// pKey := secp256k1.GenPrivKey()
			address, err := bech32.ConvertAndEncode(stortypes.AddressPrefix, pKey.PubKey().Address().Bytes())
			if err != nil {
				return err
			}

			keyExport := crypto.StorPrivKey{
				Address: address,
				Key:     crypto.ExportPrivKey(pKey),
			}

			alreadyKey := crypto.KeyExists(clientCtx)
			buf := bufio.NewReader(cmd.InOrStdin())

			if alreadyKey {
				yes, err := input.GetConfirmation("Key already exists, would you like to overwrite it? If so please make sure you have created a backup.", buf, cmd.ErrOrStderr())
				if err != nil {
					return err
				}

				if !yes {
					return nil
				}
			}

			err = crypto.WriteKey(clientCtx, &keyExport)
			if err != nil {
				return err
			}

			fmt.Printf("Your new address is %s\n", address)

			return nil
		},
	}

	return cmd
}
