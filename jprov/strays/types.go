package strays

import (
	"math/rand"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

type StrayManager struct {
    Address       string
    Context       *utils.Context
    ClientContext client.Context
    Cmd           *cobra.Command
    Db archive.ArchiveDB
    hands         []*LittleHand
    Waiter        sync.WaitGroup
    Strays        []*types.Strays
    Ip            string
    Provider types.Providers
    Rand          *rand.Rand
}

func NewStrayManager(cmd *cobra.Command, db archive.ArchiveDB) *StrayManager {
	clientCtx := client.GetClientContextFromCmd(cmd)
	ctx := utils.GetServerContextFromCmd(cmd)

	addr, err := crypto.GetAddress(clientCtx) // Getting the address of the provider to compare it to the strays.
	if err != nil {
		ctx.Logger.Error(err.Error())
		return nil
	}
	qClient := types.NewQueryClient(clientCtx)

	req := types.QueryProviderRequest{ // Ask the network what my own IP address is registered to.
		Address: addr,
	}

	provs, err := qClient.Providers(cmd.Context(), &req) // Publish the ask.
	if err != nil {
		ctx.Logger.Error(err.Error())
		return nil
	}
	ip := provs.Providers.Address // Our IP address

	return &StrayManager{
		Address:       addr,
        Context:       ctx,
        ClientContext: clientCtx,
        Cmd:           cmd,
        Db: db,
		hands:         []*LittleHand{},
        Ip:            ip,
        Provider: provs.Providers,
		Strays:        []*types.Strays{},
	}
}

type LittleHand struct {
	Stray         *types.Strays
	Waiter        *sync.WaitGroup
	Database      archive.ArchiveDB
	Busy          bool
	Cmd           *cobra.Command
	ClientContext client.Context
	Id            uint
	Address       string
}
