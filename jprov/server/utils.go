package server

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

const ErrNotYours = "not your deal"

func queryBlock(clientCtx *client.Context, cid string) (string, error) {
	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryActiveDealRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return "", err
	}

	return res.ActiveDeals.Blocktoprove, nil
}

func checkVerified(clientCtx *client.Context, cid string, self string) (bool, error) {
	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryActiveDealRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return false, err
	}

	ver, err := strconv.ParseBool(res.ActiveDeals.Proofverified)
	if err != nil {
		return false, err
	}

	if res.ActiveDeals.Provider != self {
		return false, fmt.Errorf(ErrNotYours)
	}

	return ver, nil
}
