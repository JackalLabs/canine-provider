package server

import (
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/spf13/cobra"
)

type serverContext struct {
	messageSize int
	gasCap      int
	address     string
	cosmosCtx   client.Context
	provider    storageTypes.Providers
}

func newServerContext(cmd *cobra.Command) (*serverContext, error) {
	cosmosCtx := client.GetClientContextFromCmd(cmd)
	address, err := crypto.GetAddress(cosmosCtx)
	if err != nil {
		return nil, err
	}

	return &serverContext{
		address:   address,
		cosmosCtx: cosmosCtx,
	}, nil
}
