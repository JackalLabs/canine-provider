package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/wealdtech/go-merkletree"
)

const ErrNotYours = "not your deal"

func HashData(cmd *cobra.Command, fid string) (string, string, error) {
	clientCtx := client.GetClientContextFromCmd(cmd)
	ctx := utils.GetServerContextFromCmd(cmd)
	path := utils.GetStoragePath(clientCtx, fid)
	files, err := os.ReadDir(filepath.Clean(path))
	if err != nil {
		ctx.Logger.Error(err.Error())
	}
	size := 0
	var list [][]byte

	for i := 0; i < len(files); i++ {

		path := filepath.Join(utils.GetStoragePath(clientCtx, fid), fmt.Sprintf("%d.jkl", i))

		dat, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			ctx.Logger.Error(err.Error())
		}

		size = size + len(dat)

		h := sha256.New()
		_, err = io.WriteString(h, fmt.Sprintf("%d%x", i, dat))
		if err != nil {
			return "", "", err
		}
		hashName := h.Sum(nil)

		list = append(list, hashName)

	}

	t, err := merkletree.New(list)
	if err != nil {
		ctx.Logger.Error(err.Error())
	}

	return hex.EncodeToString(t.Root()), fmt.Sprintf("%d", size), nil
}

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
