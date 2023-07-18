package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
)

func Combine(dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, src)
	return err
}

// Get all block filename in directory
// Error is returned if the directory contains more directory
func GetFilenames(dir string) (filenames []string, err error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			err = errors.New("this directory have another directory")
			return
		}
		filenames = append(filenames, dir.Name())
	}

	return
}

// Glue all blocks for single fid
func GlueAllBlocks(ctx client.Context, fid string) error {

	return nil
}

// Create a file at newName and put all contents of blockNames in order
func glueAllFiles(blockNames []string, newName string) (f *os.File, err error) {
	f, err = os.Create(newName)
	return
}

// Get block index from fileName
// The fileName format should be in: i.jkl where i is an index
func getIndex(filename string) (index int, err error) {
	strIndex, ok := strings.CutSuffix(filename, ".jkl")
	if !ok {
		err = fmt.Errorf("invalid block file name: %s", filename)
		return
	}

	index, err = strconv.Atoi(strIndex)
	return
}
