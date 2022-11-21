package utils

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func DownloadFileFromURL(cmd *cobra.Command, url string, fid string, cid string, db *leveldb.DB) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/d/%s", url, fid))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, resp.Body)
	if err != nil {
		return "", err
	}

	reader := bytes.NewReader(buff.Bytes())

	hashName, err := WriteFileToDisk(cmd, reader, reader, nil, size)
	if err != nil {
		return hashName, err
	}

	err = SaveToDatabase(hashName, cid, db)
	if err != nil {
		return hashName, err
	}

	return hashName, nil
}

func WriteFileToDisk(cmd *cobra.Command, reader io.Reader, file io.ReaderAt, closer io.Closer, size int64) (string, error) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return "", err
	}

	clientCtx := client.GetClientContextFromCmd(cmd)

	h := sha256.New()
	_, err = io.Copy(h, reader)
	if err != nil {
		return "", err
	}
	hashName := h.Sum(nil)
	fid, err := MakeFid(hashName)
	if err != nil {
		return "", err
	}

	path := GetStoragePath(clientCtx, fid)

	// This is path which we want to store the file
	direrr := os.MkdirAll(path, os.ModePerm)
	if direrr != nil {
		return fid, direrr
	}

	var blocksize int64 = 1024
	var i int64
	for i = 0; i < size; i += blocksize {
		f, err := os.OpenFile(filepath.Join(path, fmt.Sprintf("%d.jkl", i/blocksize)), os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return fid, err
		}

		firstx := make([]byte, blocksize)
		read, err := file.ReadAt(firstx, i)
		if debug {
			fmt.Printf("Bytes read: %d", read)
		}
		if err != nil && err != io.EOF {
			return fid, err
		}
		firstx = firstx[:read]
		_, writeerr := f.Write(firstx)
		if writeerr != nil {
			return fid, err
		}
		f.Close()
	}
	if closer != nil {
		closer.Close()
	}
	return fid, nil
}

func SaveToDatabase(fid string, strcid string, db *leveldb.DB) error {
	err := db.Put(MakeDowntimeKey(strcid), []byte(fmt.Sprintf("%d", 0)), nil)
	if err != nil {
		fmt.Printf("Downtime Database Error: %v\n", err)
		return err
	}
	derr := db.Put(MakeFileKey(strcid), []byte(fid), nil)
	if derr != nil {
		fmt.Printf("File Database Error: %v\n", derr)
		return err
	}

	fmt.Printf("%s %s\n", fid, "Added to database")

	_, cerr := db.Get(MakeFileKey(strcid), nil)
	if cerr != nil {
		fmt.Printf("Hash Database Error: %s\n", cerr.Error())
		return err
	}

	return nil
}
