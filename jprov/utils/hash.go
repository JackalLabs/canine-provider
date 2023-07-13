package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

func MakeFID(reader io.Reader) (string, error) {
	h := sha256.New()
	_, err := io.Copy(h, reader)
	if err != nil {
		return "", err
	}
	hashName := h.Sum(nil)
	fid, err := MakeFid(hashName)
	if err != nil {
		return "", err
	}

	return fid, nil
}

func SaveFileToDisk(cmd *cobra.Command, fid string, file io.Reader) error {
	clientCtx := client.GetClientContextFromCmd(cmd)

	// creating file path
	path := GetStoragePathV2(clientCtx, fid)
	f, dirErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o666)
	if dirErr != nil {
		return dirErr
	}
	defer f.Close()

	_, err := io.Copy(f, file) // writing file to disk
	if err != nil {
		return err
	}

	return nil
}

func WriteTreeToDisk(cmd *cobra.Command, fid string, tree *merkletree.MerkleTree) error {
	clientCtx := client.GetClientContextFromCmd(cmd)

	exportedTree, err := tree.Export()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(GetStoragePathForTree(clientCtx, fid), os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		return err
	}
	_, err = f.Write(exportedTree)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

// This function generates merkletree from file.
// The stream of bytes read from file is divided into blocks by blockSize.
// Thus, smaller blockSize will increase the size of the tree but decrease the size of proof
// and vice versa.
func CreateMerkleTree(blockSize, fileSize int64, file io.Reader) (*merkletree.MerkleTree, error) {
	data := make([][]byte, fileSize/blockSize+1)

	// Divide files into blocks
	for i := int64(0); i < fileSize; i += blockSize {
		firstX := make([]byte, blockSize)
		read, err := file.Read(firstX)

		if err != nil && err != io.EOF {
			return nil, err
		}

		firstX = firstX[:read]

		// Building a block
		var hashBuilder strings.Builder
		hashBuilder.WriteString(strconv.FormatInt(i/blockSize, 10))
		hashBuilder.WriteString(hex.EncodeToString(firstX))

		// Encrypt & reduce size of data
		hash := sha256.New()
		_, err = io.WriteString(hash, hashBuilder.String())
		if err != nil {
			return nil, err
		}
		hashName := hash.Sum(nil)

		data[i/blockSize] = hashName
	}

	tree, err := merkletree.NewUsing(data, sha3.New512(), false)
	if err != nil {
		return tree, err
	}

	return tree, nil
}
