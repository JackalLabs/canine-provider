package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

const (
	FileKey     = "FILE-"
	DowntimeKey = "DWNT-"
)

func MakeFileKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", FileKey, cid))
}

func MakeDowntimeKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", DowntimeKey, cid))
}

func GetStoragePath(ctx client.Context, fid string) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")
	configFilePath := filepath.Join(configPath, fid)

	return configFilePath
}

func GetStoragePathForPiece(ctx client.Context, fid string, index int) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")
	configFilePath := filepath.Join(configPath, fid, fmt.Sprintf("%d.jkl", index))

	return configFilePath
}

func GetStoragePathForTree(ctx client.Context, fid string) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")
	configFilePath := filepath.Join(configPath, fmt.Sprintf("%s.tree", fid))

	return configFilePath
}

func GetStorageAllPath(ctx client.Context) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")

	return configPath
}

func GetDataPath(ctx client.Context) string {
	dataPath := filepath.Join(ctx.HomeDir, "data")

	return dataPath
}

func BuildAndSaveTree(cmd *cobra.Command, fid string, size int64, blockSize int64) (string, error) {
	clientCtx := client.GetClientContextFromCmd(cmd)
	path := GetStoragePath(clientCtx, fid)
	data := make([][]byte, size/blockSize+1)

	for i := int64(0); i < size; i += blockSize {
		f, err := os.OpenFile(filepath.Join(path, fmt.Sprintf("%d.jkl", i/blockSize)), os.O_RDONLY, 0o666)
		if err != nil {
			return "", err
		}

		b := make([]byte, blockSize)
		_, err = f.Read(b)
		if err != nil {
			return "", err
		}

		hash := sha256.New()
		_, err = io.WriteString(hash, fmt.Sprintf("%d%x", i/blockSize, b))
		if err != nil {
			return "", err
		}
		hashName := hash.Sum(nil)
		data[i/blockSize] = hashName

	}

	tree, err := merkletree.NewUsing(data, sha3.New512(), false)
	if err != nil {
		return "", err
	}

	exportedTree, err := tree.Export()
	if err != nil {
		return "", err
	}

	f, err := os.OpenFile(GetStoragePathForTree(clientCtx, fid), os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		return "", err
	}
	_, err = f.Write(exportedTree)
	if err != nil {
		return "", err
	}
	err = f.Close()
	if err != nil {
		return "", err
	}
	r := hex.EncodeToString(tree.Root())

	// nolint
	tree = nil
	exportedTree = nil

	runtime.GC()

	return r, nil
}

const ErrNotYours = "not your deal"

func QueryBlock(clientCtx *client.Context, cid string) (string, error) {
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

func CheckVerified(clientCtx *client.Context, cid string, self string) (bool, error) {
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
