package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
)

// File structure for storage
// Files with suffix ".jkl" contains actual data
// Files with suffix ".tree" contains exported merkle tree of the file
//
// ctx.HomeDir
//	/ storage
//		/ fid1 *directory that contains everything for fid1
//			/ fid1.jkl
//			/ fid1.tree
//		/ fid2
//			/ fid2.jkl
//			/ fid2.tree

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
	return filepath.Join(GetStorageRootDir(ctx), GetStorageFileName(fid))
}

func GetStorageRootDir(ctx client.Context) string {
	return filepath.Join(ctx.HomeDir, "storage")
}

func GetStorageFileName(fid string) string {
	return fid
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

// GetContentsPath returns file path for the file that stores the user uploaded contents
// e.g. ~/.provider/storage/<fid>/<fid>.jkl
func GetContentsPath(ctx client.Context, fid string) string {
	return filepath.Join(GetStorageRootDir(ctx), fid, GetStorageFileName(fid))		
}

// GetStoragePathForTree returns full path for fid merkle tree file
// e.g. ~/.provider/storage/<fid>/<fid>.tree
func GetStoragePathForTree(ctx client.Context, fid string) string {
	return filepath.Join(GetStorageDirForTree(ctx), fid, GetFileNameForTree(fid))
}

func GetStorageDirForTree(ctx client.Context) string {
	return filepath.Join(ctx.HomeDir, "storage")
}

func GetFileNameForTree(fid string) string {
	return fmt.Sprintf("%s.tree", fid)
}



func GetStorageAllPath(ctx client.Context) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")

	return configPath
}

func GetDataPath(ctx client.Context) string {
	dataPath := filepath.Join(ctx.HomeDir, "data")

	return dataPath
}
