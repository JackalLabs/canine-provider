package utils

import (
	"errors"
	"io"
	"os"
	"strings"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
)

// postGlueCheck verifies the result of glueing was successful by generating fid
// of the glued file and check against passed fid
func postGlueCheck(ctx client.Context, fid string) (pass bool, err error) {
	file, err := os.Open(GetContentsPath(ctx, fid))
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
func DiscoverFids(ctx client.Context) (fids []string, err error) {
	dirs, err := os.ReadDir(GetStorageRootDir(ctx))
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
func GlueAllBlocks(ctx client.Context, fid string) error {
	fileNames, err := GetBlockFileNames(GetStoragePath(ctx, fid))
	if err != nil {
		return err
	}

	ok := checkAllFileNames(fileNames)
	if !ok {
		return errors.New("invalid file structure for file storage")
	}

	err = glueAllBlocks(len(fileNames), fid)

	return err
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
		fileNames = append(fileNames, d.Name())
	}

	return
}

// glue all blocks starting from 1.jkl to <blocksCount>.jkl
func glueAllBlocks(blocksCount int, newFileName string) (err error) {
	f, err := os.Create(newFileName)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	// glue files in order
	for i := 1; i < blocksCount+1; i++ {
		if err := combine(f, getBlockFileName(i)); err != nil {
			return err
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
		errors.Join(err, src.Close())
	}()

	_, err = io.Copy(dst, src)
	return err
}

func checkAllFileNames(fileNames []string) (ok bool) {
	for _, name := range fileNames {
		if ok = checkFileName(name); !ok {
			return
		}
	}

	return true
}

// create file name for a block
func getBlockFileName(index int) string {
	var name strings.Builder
	_, _ = name.WriteString(strconv.Itoa(index))// returns length of s and a nil err
	_, _ = name.WriteString(".jkl")// returns length of s and a nil err

	return name.String()
}

// Check if the file name is a valid block file name
// The fileName format should be in: i.jkl where i is an index
func checkFileName(filename string) bool {
	strIndex, ok := strings.CutSuffix(filename, ".jkl")
	if !ok {
		return ok
	}

	_, err := strconv.Atoi(strIndex)
	if err != nil {
		return false
	}

	return true
}
