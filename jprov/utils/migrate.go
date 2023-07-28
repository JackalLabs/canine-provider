package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
)

func Migrate(ctx client.Context) {
	fids, err := DiscoverFids(ctx.HomeDir)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	for i, fid := range fids {
		fmt.Printf("\033[2K\rGlueing %d/%d files...", i, len(fids))
		err := GlueAllBlocks(ctx.HomeDir, fid)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			return
		}
		ok, err := postGlueCheck(ctx, fid)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			return
		}
		if !ok {
			fmt.Printf("Check failure: %s is corrupted\n", fid)
			return
		}
	}
	fmt.Printf("\n")
	fmt.Println("Migration finished")
}

// postGlueCheck verifies the result of glueing was successful by generating fid
// of the glued file and check against passed fid
func postGlueCheck(ctx client.Context, fid string) (pass bool, err error) {
	file, err := os.Open(GetContentsPath(ctx.HomeDir, fid))
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	resultFid, err := MakeFID(file, file)

	pass = fid == resultFid

	return
}

// DiscoverFids reads all directory entry of the storage and returns fids
func DiscoverFids(homeDir string) (fids []string, err error) {
	dirs, err := os.ReadDir(getStorageRootDir(homeDir))
	if err != nil {
		return
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			fids = append(fids, dir.Name())
		}
	}

	return
}

// Glue all blocks for single fid
func GlueAllBlocks(homeDir, fid string) error {
	fileNames, err := GetBlockFileNames(getStoragePath(homeDir, fid))
	if err != nil {
		return err
	}

	err = glueAllBlocks(homeDir, fid, len(fileNames))

	return err
}

// glue all blocks starting from 1.jkl to <blocksCount>.jkl
func glueAllBlocks(homeDir, fid string, blocksCount int) (err error) {
	f, err := os.Create(GetContentsPath(homeDir, fid))
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	// glue files in order
	for i := 0; i < blocksCount; i++ {
		path := filepath.Join(GetFidDir(homeDir, fid), getBlockFileName(i))
		if err := combine(f, path); err != nil {
			return err
		}
	}

	return
}

// Get all files' name in directory
// An error is returned if the directory contains more directory
func GetBlockFileNames(dir string) (fileNames []string, err error) {
	dirEntry, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, d := range dirEntry {
		if d.IsDir() {
			err = errors.New("this directory have another directory")
			return
		}

		ok := checkFileName(d.Name())
		if ok {
			fileNames = append(fileNames, d.Name())
		}
	}

	return
}

// combine opens source file and copy its contents into destination
func combine(dst io.Writer, srcFileName string) error {
	src, err := os.Open(srcFileName)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, src.Close())
	}()

	_, err = io.Copy(dst, src)
	return err
}

// Check if the file name is a valid block file name
// The fileName format should be in: i.jkl where i is an index
func checkFileName(filename string) bool {
	strIndex, ok := strings.CutSuffix(filename, ".jkl")
	if !ok {
		return ok
	}

	_, err := strconv.Atoi(strIndex)
	return err == nil
}

// Legacy file paths
func getStoragePath(homeDir, fid string) string {
	configPath := filepath.Join(homeDir, "storage")
	configFilePath := filepath.Join(configPath, fid)

	return configFilePath
}

// create file name for a block
func getBlockFileName(index int) string {
	var name strings.Builder
	_, _ = name.WriteString(strconv.Itoa(index)) // returns length of s and a nil err
	_, _ = name.WriteString(".jkl")              // returns length of s and a nil err

	return name.String()
}
