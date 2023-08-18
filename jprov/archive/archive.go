package archive

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	merkletree "github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

const FilePerm os.FileMode = 0o666

type Archive interface {
	// WriteToDisk creates a directory and file to store the file
	// Returns bytes written and nil error if write was successful
	// Returns non-nil error when it fails to create directory or create and write file
	WriteFileToDisk(data io.Reader, fid string) (written int64, err error)
	// GetPiece returns a piece of block at index of a file
	GetPiece(fid string, index, blockSize int64) (block []byte, err error)
	// RetrieveFile returns an io.ReaderCloser for the file data
	// The data must be closed after reading is done
	// Error is returned when such file does not exist
	RetrieveFile(fid string) (data io.ReadCloser, err error)
	// WriteTreeToDisk creates a directory and file to store merkle tree
	// Returns error if the process fails
	WriteTreeToDisk(fid string, tree *merkletree.MerkleTree) (err error)
	// RetrieveTree returns *merkletree
	// Returns error if the tree is not found
	RetrieveTree(fid string) (tree *merkletree.MerkleTree, err error)
}

var _ Archive = &SingleCellArchive{}

type SingleCellArchive struct {
	rootDir string
	pathFactory pathFactory
}

func (f *SingleCellArchive) WriteFileToDisk(data io.Reader, fid string) (written int64, err error) {
	path := f.pathFactory.FilePath(fid)
	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, FilePerm)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	} ()

	written, err = io.Copy(file, data)
	return
}

func (f *SingleCellArchive) GetPiece(fid string, index, blockSize int64) (block []byte, err error) {
	file, err := os.Open(f.pathFactory.FilePath(fid))
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	block = make([]byte, blockSize)
	n, err := file.ReadAt(block, index*blockSize)
	// ignoring io.EOF with n > 0 because the file size is not always n * blockSize
	if (err != nil && err != io.EOF) || (err == io.EOF && n == 0) {
		return
	}

	return block, nil
}

func (f *SingleCellArchive) RetrieveFile(fid string) (data io.ReadCloser, err error) {
	data, err = os.Open(f.pathFactory.FilePath(fid))
	return
}

func (f *SingleCellArchive) WriteTreeToDisk(fid string, tree *merkletree.MerkleTree) (err error){
	path := f.pathFactory.TreePath(fid)
	err = os.Mkdir(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, FilePerm)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	data, err := tree.Export()
	if err != nil {
		return
	}

	_, err = file.Write(data)
	return
}

func (f *SingleCellArchive) RetrieveTree(fid string) (tree *merkletree.MerkleTree, err error) {
	rawTree, err := os.ReadFile(f.pathFactory.TreePath(fid))
	if err != nil {
		return
	}

	tree, err = merkletree.ImportMerkleTree(rawTree, sha3.New512())
	return
}
