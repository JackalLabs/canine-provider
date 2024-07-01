package server

import (
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

type serverContext struct {
	address   string
	cosmosCtx client.Context
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
