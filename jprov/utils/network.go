package utils

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func DownloadFileFromURL(cmd *cobra.Command, url string, fid string, cid string, db *leveldb.DB) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("%s/d/%s", url, fid))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, resp.Body)
	if err != nil {
		return nil, err
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

func WriteFileToDisk(cmd *cobra.Command, reader io.Reader, file io.ReaderAt, closer io.Closer, size int64) ([]byte, error) {
	h := sha256.New()
	_, err := io.Copy(h, reader)
	if err != nil {
		return nil, err
	}
	hashName := h.Sum(nil)

	files, err := cmd.Flags().GetString(types.DataDir)
	if err != nil {
		return nil, err
	}

	// This is path which we want to store the file
	direrr := os.MkdirAll(fmt.Sprintf("%s/networkfiles/%s/", files, fmt.Sprintf("%x", hashName)), os.ModePerm)
	if direrr != nil {
		return hashName, direrr
	}

	var blocksize int64 = 1024
	var i int64
	for i = 0; i < size; i += blocksize {
		f, err := os.OpenFile(fmt.Sprintf("%s/networkfiles/%s/%d%s", files, fmt.Sprintf("%x", hashName), i/blocksize, ".jkl"), os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return hashName, err
		}

		firstx := make([]byte, blocksize)
		read, err := file.ReadAt(firstx, i)
		fmt.Println(read)
		if err != nil && err != io.EOF {
			return hashName, err
		}
		firstx = firstx[:read]
		// fmt.Printf(": %s :\n", string(firstx))
		read, writeerr := f.Write(firstx)
		fmt.Println(read)
		if writeerr != nil {
			return hashName, err
		}
		f.Close()
	}
	if closer != nil {
		closer.Close()
	}
	return hashName, nil
}

func SaveToDatabase(hashName []byte, strcid string, db *leveldb.DB) error {
	err := db.Put(MakeDowntimeKey(strcid), []byte(fmt.Sprintf("%d", 0)), nil)
	if err != nil {
		fmt.Printf("Downtime Database Error: %v\n", err)
		return err
	}
	derr := db.Put(MakeFileKey(strcid), []byte(fmt.Sprintf("%x", hashName)), nil)
	if derr != nil {
		fmt.Printf("File Database Error: %v\n", derr)
		return err
	}

	fmt.Printf("%s %s\n", fmt.Sprintf("%x", hashName), "Added to database")

	_, cerr := db.Get(MakeFileKey(strcid), nil)
	if cerr != nil {
		fmt.Printf("Hash Database Error: %s\n", cerr.Error())
		return err
	}

	return nil
}
