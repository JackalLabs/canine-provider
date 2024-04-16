package archive

import (
	"errors"
	"fmt"
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
	RetrieveFile(fid string) (data io.ReadSeekCloser, err error)
	// Only use this for claiming strays check
	FileExist(fid string) bool
	// WriteTreeToDisk creates a directory and file to store merkle tree
	// Returns error if the process fails
	WriteTreeToDisk(fid string, tree *merkletree.MerkleTree) (err error)
	// RetrieveTree returns *merkletree
	// Returns error if the tree is not found
	RetrieveTree(fid string) (tree *merkletree.MerkleTree, err error)
	// Delete deletes archive from disk. This include the file and merkle tree.
	Delete(fid string) error
}

var _ Archive = &SingleCellArchive{}

var _ Archive = &HybridCellArchive{}

type HybridCellArchive struct {
	rootDir     string
	pathFactory *SingleCellPathFactory
}

func NewHybridCellArchive(rootDir string) *HybridCellArchive {
	return &HybridCellArchive{
		rootDir:     rootDir,
		pathFactory: NewSingleCellPathFactory(rootDir),
	}
}

func (f *HybridCellArchive) WriteFileToDisk(data io.Reader, fid string) (written int64, err error) {
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
	}()

	written, err = io.Copy(file, data)
	return
}

func (h *HybridCellArchive) getLegacyPiece(file *os.File, blockSize int64) ([]byte, error) {
	block := make([]byte, blockSize)

	_, err := file.Read(block)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (h *HybridCellArchive) GetPiece(fid string, index, blockSize int64) (block []byte, err error) {
	file, err := os.Open(filepath.Join(h.rootDir, "storage", fid, fmt.Sprintf("%d.jkl", index)))
	if err == nil {
		// legacy file system
		defer func() {
			err = errors.Join(err, file.Close())
		}()
		return h.getLegacyPiece(file, blockSize)
	} else if !os.IsNotExist(err) { // unkown error
		return nil, err
	}

	file, err = os.Open(h.pathFactory.FilePath(fid))
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	block = make([]byte, blockSize)
	n, err := file.ReadAt(block, index*blockSize)
	// ignoring io.EOF with n > 0 because the file size is not always n * blockSize
	if (err != nil && err != io.EOF) || (err == io.EOF && n == 0) {
		return nil, err
	}

	block = block[:n]
	return block, nil
}

func (h *HybridCellArchive) RetrieveFile(fid string) (data io.ReadSeekCloser, err error) {
	data, err = os.Open(h.pathFactory.FilePath(fid))
	return
}

func (h *HybridCellArchive) FileExist(fid string) bool {
	_, err := os.Stat(h.pathFactory.FilePath(fid))
	return errors.Is(err, os.ErrNotExist)
}

func (h *HybridCellArchive) WriteTreeToDisk(fid string, tree *merkletree.MerkleTree) (err error) {
	path := h.pathFactory.TreePath(fid)
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
	}()

	data, err := tree.Export()
	if err != nil {
		return
	}

	_, err = file.Write(data)
	return
}

func (h *HybridCellArchive) retrieveLegacyTree(fid string) (*merkletree.MerkleTree, error) {
    return nil, nil
}
func (h *HybridCellArchive) RetrieveTree(fid string) (tree *merkletree.MerkleTree, err error) {
	rawTree, err := os.ReadFile(h.pathFactory.TreePath(fid))
	if err != nil {
		return
	}

	tree, err = merkletree.ImportMerkleTree(rawTree, sha3.New512())
	return
}

func (h *HybridCellArchive) Delete(fid string) error {
	// since the file and merkle tree is saved together in an isolated directory,
	// just delete the whole directory
	err := os.RemoveAll(h.pathFactory.FileDir(fid))
	if err != nil {
		// filePath factory might be broken
		// read os.RemoveAll error conditions at std doc
		return err
	}
	return nil
}

type SingleCellArchive struct {
	rootDir     string
	pathFactory *SingleCellPathFactory
}

func NewSingleCellArchive(rootDir string) *SingleCellArchive {
	return &SingleCellArchive{
		rootDir:     rootDir,
		pathFactory: NewSingleCellPathFactory(rootDir),
	}
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
	}()

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

	block = block[:n]
	return block, nil
}

func (f *SingleCellArchive) RetrieveFile(fid string) (data io.ReadSeekCloser, err error) {
	data, err = os.Open(f.pathFactory.FilePath(fid))
	return
}

func (f *SingleCellArchive) FileExist(fid string) bool {
	_, err := os.Stat(f.pathFactory.FilePath(fid))
	return errors.Is(err, os.ErrNotExist)
}

func (f *SingleCellArchive) WriteTreeToDisk(fid string, tree *merkletree.MerkleTree) (err error) {
	path := f.pathFactory.TreePath(fid)
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

func (f *SingleCellArchive) Delete(fid string) error {
	// since the file and merkle tree is saved together in an isolated directory,
	// just delete the whole directory
	return os.RemoveAll(f.pathFactory.FileDir(fid))
}
