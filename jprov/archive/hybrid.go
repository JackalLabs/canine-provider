package archive

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	merkletree "github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

var _ Archive = &HybridCellArchive{}

var multicellFileNameRegex *regexp.Regexp = regexp.MustCompile("[0-9]+.jkl")

type HybridCellArchive struct {
	rootDir           string
	pathFactory       *SingleCellPathFactory
	legacyPathFactory *MultiCellPathFactory
}

func NewHybridCellArchive(rootDir string) *HybridCellArchive {
	return &HybridCellArchive{
		rootDir:           rootDir,
		pathFactory:       NewSingleCellPathFactory(rootDir),
		legacyPathFactory: NewMultiCellPathFactory(rootDir),
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
	if os.IsNotExist(err) {
		//try glueing
		err = h.glueBlocks(fid)
		if err != nil {
			return nil, err
		}
		return os.Open(h.pathFactory.FilePath(fid))
	}
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
	rawTree, err := os.ReadFile(h.legacyPathFactory.TreePath(fid))
	if err != nil {
		return nil, err
	}

	tree, err := merkletree.ImportMerkleTree(rawTree, sha3.New512())
	return tree, err
}

func (h *HybridCellArchive) RetrieveTree(fid string) (tree *merkletree.MerkleTree, err error) {
	tree, err = h.retrieveLegacyTree(fid) // attempt to get legacy
	if err == nil {
		return tree, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

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

func (h *HybridCellArchive) glueBlocks(fid string) error {
	blocks, err := h.lastBlock(fid)
	if err != nil {
		return err
	}

	tmpFid := "tmp-" + fid
	tmpFile := filepath.Join(h.pathFactory.FileDir(fid), h.pathFactory.fileName(tmpFid))
	if _, err := os.Stat(tmpFile); err == nil {
		return errors.New("file is being glued together")
	}

	file, err := os.OpenFile(tmpFile, os.O_WRONLY|os.O_CREATE, FilePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < blocks; i++ {
		err = combine(file, h.legacyPathFactory.PiecePath(fid, i))
		if err != nil {
			return err
		}
	}

	file.Close()
	return os.Rename(tmpFile, h.pathFactory.FilePath(fid))
}

// combine opens source file and copy its contents into destination
func combine(dst io.Writer, srcFileName string) error {
	src, err := os.Open(srcFileName)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, src.Close())
	}()

	_, err = io.Copy(dst, src)
	return err
}

func (h *HybridCellArchive) lastBlock(fid string) (int, error) {
	dir, err := os.Open(h.pathFactory.FileDir(fid))
	if err != nil {
		return -1, err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(0)
	if err != nil {
		return -1, err
	}

	last := -1
	for _, f := range files {
		if multicellFileNameRegex.Match([]byte(f)) {
			last++
		}
	}

	return last, nil
}
