package utils

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tendermint/tendermint/libs/log"
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

	hashName, _, err := WriteFileToDisk(cmd, reader, reader, nil, size, db, logger)
	if err != nil {
		return "", err
	}

	return hashName, nil
}

func WriteFileToDisk(cmd *cobra.Command, reader io.Reader, file io.ReaderAt, closer io.Closer, size int64, db *leveldb.DB, logger log.Logger) (string, string, error) {
	blockSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		return "", "", err
	}

	clientCtx := client.GetClientContextFromCmd(cmd)

	h := sha256.New()
	_, err = io.Copy(h, reader)
	if err != nil {
		return "", "", err
	}
	hashName := h.Sum(nil)
	fid, err := MakeFid(hashName)
	if err != nil {
		return "", "", err
	}

	h = nil // marking for removal from gc

	path := GetStoragePath(clientCtx, fid)

	// This is path which we want to store the file
	direrr := os.MkdirAll(path, os.ModePerm)
	if direrr != nil {
		return fid, "", direrr
	}

	for i := int64(0); i < size; i += blockSize {
		f, err := os.OpenFile(filepath.Join(path, fmt.Sprintf("%d.jkl", i/blockSize)), os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return fid, "", err
		}

		firstx := make([]byte, blockSize)
		read, err := file.ReadAt(firstx, i)
		logger.Debug(fmt.Sprintf("Bytes read: %d", read))

		if err != nil && err != io.EOF {
			return fid, "", err
		}
		firstx = firstx[:read]
		_, writeErr := f.Write(firstx)
		if writeErr != nil {
			return fid, "", err
		}
		err = f.Close()
		if err != nil {
			return fid, "", err
		}

		hash := sha256.New()
		_, err = io.WriteString(hash, fmt.Sprintf("%d%x", i/blockSize, firstx))
		if err != nil {
			return fid, "", err
		}
	}
	if closer != nil {
		err := closer.Close()
		if err != nil {
			return fid, "", err
		}
	}

	GetServerContextFromCmd(cmd).Logger.Info("Starting merkle tree construction...")

	r, err := BuildAndSaveTree(cmd, fid, size, blockSize)
	if err != nil {
		return fid, "", err
	}

	return fid, r, nil
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
