package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

// MakeFID generates fid from the data it reads from reader
func MakeFID(reader io.Reader, seeker io.Seeker) (fid string, err error) {
	current, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return
	}
	_, err = seeker.Seek(0, io.SeekStart)
	if err != nil {
		return
	}

	h := sha256.New()
	_, err = io.Copy(h, reader)
	if err != nil {
		return "", err
	}

	hashName := h.Sum(nil)
	fid, err = MakeFid(hashName)
	if err != nil {
		return "", err
	}

	_, err = seeker.Seek(current, io.SeekStart)
	if err != nil {
		return
	}
	return fid, nil
}

// This function generates merkletree from file.
// The stream of bytes read from file is divided into blocks by blockSize.
// Thus, smaller blockSize will increase the size of the tree but decrease the size of proof
// and vice versa.
func CreateMerkleTree(blockSize, fileSize int64, file io.Reader, seeker io.Seeker) (t *merkletree.MerkleTree, err error) {
	current, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return
	}
	_, err = seeker.Seek(0, io.SeekStart)
	if err != nil {
		return
	}

	data := make([][]byte, fileSize/blockSize+1)

	// Divide files into blocks
	for i := int64(0); i < fileSize; i += blockSize {
		firstX := make([]byte, blockSize)
		//read, err := file.Read(firstX)
		_, err := file.Read(firstX)

		if err != nil && err != io.EOF {
			return nil, err
		}

		//firstX = firstX[:read]

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

	_, err = seeker.Seek(current, io.SeekStart)
	if err != nil {
		return
	}

	t, err = merkletree.NewUsing(data, sha3.New512(), false)
	return
}

// GetBlock returns the block at index of file.
// The blockSize must be the same as when the file's merkle tree was created.
func GetBlock(index, blockSize int, file *os.File) (block []byte, err error) {
	offset := blockSize * index
	if offset < 0 {
		err = errors.New("index and blockSize can't be negative int")
		return
	}

	current, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return
	}

	buf := make([]byte, blockSize)

	_, err = file.ReadAt(buf, int64(offset))
	if err != nil {
		return
	}

	_, err = file.Seek(current, io.SeekStart)
	return
}
