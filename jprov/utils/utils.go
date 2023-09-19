package utils

import (
	"fmt"
	"path/filepath"

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

func getStorageRootDir(homeDir string) string {
	return filepath.Join(homeDir, "storage")
}

func GetContentsFileName(fid string) string {
	return fid
}

func GetFidDir(homeDir, fid string) string {
	return filepath.Join(getStorageRootDir(homeDir), fid)
}

// GetContentsPath returns file path for the file that stores the user uploaded contents
// e.g. ~/.provider/storage/<fid>/<fid>.jkl
func GetContentsPath(homeDir, fid string) string {
	return filepath.Join(GetFidDir(homeDir, fid), GetContentsFileName(fid))
}

// GetStoragePathForTree returns full path for fid merkle tree file
// e.g. ~/.provider/storage/<fid>/<fid>.tree
func GetStoragePathForTree(homeDir, fid string) string {
	return filepath.Join(GetFidDir(homeDir, fid), GetTreeFileName(fid))
}

func GetTreeFileName(fid string) string {
	return fmt.Sprintf("%s.tree", fid)
}

func GetStorageAllPath(ctx client.Context) string {
	configPath := filepath.Join(ctx.HomeDir, "storage")

	return configPath
}

func GetArchiveDBPath(ctx client.Context) string {
	dataPath := filepath.Join(ctx.HomeDir, "archivedb")

	return dataPath
}
func GetDowntimeDBPath(ctx client.Context) string {
	dataPath := filepath.Join(ctx.HomeDir, "downtimedb")

	return dataPath
}
