package utils

import (
	"fmt"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
)

const (
	UPTIME_LEFT_KEY = "UPTL-"
	FILE_KEY        = "FILE-"
	TREE_KEY        = "TREE-"
	DOWNTIME_KEY    = "DWNT-"
)

func MakeUptimeKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", UPTIME_LEFT_KEY, cid))
}

func MakeFileKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", FILE_KEY, cid))
}

func MakeTreeKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", TREE_KEY, cid))
}

func MakeDowntimeKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", DOWNTIME_KEY, cid))
}

func GetStoragePath(ctx client.Context, fid string) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")
	configFilePath := filepath.Join(configPath, fid)

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
