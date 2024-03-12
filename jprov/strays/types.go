package strays

import (
	"math/rand"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/spf13/cobra"
)

type StrayManager struct {
    Address       string
    Archive archive.Archive
    Context       *utils.Context
    ClientContext client.Context
    Cmd           *cobra.Command
    archivedb archive.ArchiveDB
    downtimedb *archive.DowntimeDB
    hands         []*LittleHand
    Waiter        sync.WaitGroup
    Strays        []*types.Strays
    Ip            string
    Provider types.Providers
    Rand          *rand.Rand
}

func NewStrayManager(
    cmd *cobra.Command,
    archivedb archive.ArchiveDB, 
    downtimedb *archive.DowntimeDB,
) (*StrayManager, error) {
	clientCtx := client.GetClientContextFromCmd(cmd)
	ctx := utils.GetServerContextFromCmd(cmd)

	addr, err := crypto.GetAddress(clientCtx) // Getting the address of the provider to compare it to the strays.
	if err != nil {
		ctx.Logger.Error(err.Error())
		return nil, err
	}
	qClient := types.NewQueryClient(clientCtx)

	req := types.QueryProviderRequest{ // Ask the network what my own IP address is registered to.
		Address: addr,
	}

	provs, err := qClient.Providers(cmd.Context(), &req) // Publish the ask.
	if err != nil {
		ctx.Logger.Error(err.Error())
		return nil, err
	}
	ip := provs.Providers.Address // Our IP address

    archive := archive.NewSingleCellArchive(ctx.Config.RootDir)

	return &StrayManager{
		Address:       addr,
        Context:       ctx,
        ClientContext: clientCtx,
        Cmd:           cmd,
        Archive: archive,
        archivedb: archivedb,
        downtimedb: downtimedb,
		hands:         []*LittleHand{},
        Ip:            ip,
        Provider: provs.Providers,
		Strays:        []*types.Strays{},
	}, nil
}

type LittleHand struct {
    Archive archive.Archive
	Stray         *types.Strays
	Waiter        *sync.WaitGroup
	Database      archive.ArchiveDB
	Busy          bool
	Cmd           *cobra.Command
	ClientContext client.Context
	Id            uint
	Address       string
    Logger log.Logger 
}
