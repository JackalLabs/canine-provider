package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func DownloadFileFromURL(cmd *cobra.Command, url string, fid string, cid string, db *leveldb.DB, logger log.Logger) (string, error) {
	logger.Info(fmt.Sprintf("Getting %s from %s", fid, url))

	resp, err := http.Get(fmt.Sprintf("%s/download/%s", url, fid))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to find file on network")
	}
	defer resp.Body.Close()

	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, resp.Body)
	if err != nil {
		return "", err
	}

	reader := bytes.NewReader(buff.Bytes())

	hashName, _, _, err := WriteFileToDisk(cmd, reader, reader, nil, size, db, logger)
	if err != nil {
		return "", err
	}

	return hashName, nil
}

func WriteFileToDisk(cmd *cobra.Command, reader io.Reader, file io.ReaderAt, closer io.Closer, size int64, db *leveldb.DB, logger log.Logger) (string, string, [][]byte, error) {
	blockSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		return "", "", nil, err
	}

	data := make([][]byte, size/blockSize+1)
	clientCtx := client.GetClientContextFromCmd(cmd)

	h := sha256.New()
	_, err = io.Copy(h, reader)
	if err != nil {
		return "", "", data, err
	}
	hashName := h.Sum(nil)
	fid, err := MakeFid(hashName)
	if err != nil {
		return "", "", data, err
	}

	h = nil // marking for removal from gc

	path := GetStoragePath(clientCtx, fid)

	// This is path which we want to store the file
	direrr := os.MkdirAll(path, os.ModePerm)
	if direrr != nil {
		return fid, "", data, direrr
	}

	for i := int64(0); i < size; i += blockSize {
		f, err := os.OpenFile(filepath.Join(path, fmt.Sprintf("%d.jkl", i/blockSize)), os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return fid, "", data, err
		}

		firstx := make([]byte, blockSize)
		read, err := file.ReadAt(firstx, i)

		if err != nil && err != io.EOF {
			return fid, "", data, err
		}
		firstx = firstx[:read]
		_, writeErr := f.Write(firstx)
		if writeErr != nil {
			return fid, "", data, err
		}
		err = f.Close()
		if err != nil {
			return fid, "", data, err
		}

		hash := sha256.New()
		_, err = io.WriteString(hash, fmt.Sprintf("%d%x", i/blockSize, firstx))
		if err != nil {
			return fid, "", data, err
		}
		hashName := hash.Sum(nil)
		data[i/blockSize] = hashName

	}
	if closer != nil {
		err := closer.Close()
		if err != nil {
			return fid, "", data, err
		}
	}

	GetServerContextFromCmd(cmd).Logger.Info("Starting merkle tree construction...")

	tree, err := merkletree.NewUsing(data, sha3.New512(), false)
	if err != nil {
		return fid, "", data, err
	}

	r := hex.EncodeToString(tree.Root())

	exportedTree, err := tree.Export()
	if err != nil {
		return fid, r, data, err
	}

	tree = nil // for GC

	f, err := os.OpenFile(GetStoragePathForTree(clientCtx, fid), os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		return fid, r, data, err
	}
	_, err = f.Write(exportedTree)
	if err != nil {
		return fid, r, data, err
	}
	err = f.Close()
	if err != nil {
		return fid, r, data, err
	}

	// nolint
	exportedTree = nil

	runtime.GC()

	return fid, r, data, nil
}

func SaveToDatabase(fid string, strcid string, db *leveldb.DB, logger log.Logger) error {
	err := db.Put(MakeDowntimeKey(strcid), []byte(fmt.Sprintf("%d", 0)), nil)
	if err != nil {
		logger.Error("Downtime Database Error: %v", err)
		return err
	}
	derr := db.Put(MakeFileKey(strcid), []byte(fid), nil)
	if derr != nil {
		logger.Error("File Database Error: %v", derr)
		return err
	}

	logger.Info(fmt.Sprintf("%s %s", fid, "Added to database"))

	_, cerr := db.Get(MakeFileKey(strcid), nil)
	if cerr != nil {
		logger.Error("Hash Database Error: %s", cerr.Error())
		return err
	}

	return nil
}
