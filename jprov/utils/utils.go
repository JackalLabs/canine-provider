package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
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

func GetStoragePath(ctx *Context, fid string) string {
	configPath := filepath.Join(ctx.Config.RootDir, "storage")
	configFilePath := filepath.Join(configPath, fid)

	return configFilePath
}

func GetStoragePathV2(ctx client.Context, fid string) string {
	builder := strings.Builder{}
	builder.WriteString(fid)
	builder.WriteString(".jkl")

	configPath := filepath.Join(ctx.HomeDir, "storage")
	configFilePath := filepath.Join(configPath, builder.String())

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
