package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type pathFactory interface {
	// FilePath returns a file path for fid
	FilePath(fid string) (path string)
	// TreePath returns a file path for merkle tree
	TreePath(fid string) (path string)
}

// Check if the struct satifies the inferface
var _ pathFactory = &SingleCellPathFactory{}

// SingleCellPathFactory is used for a file system that stores
// files as a single file
type SingleCellPathFactory struct {
	rootDir string
	fileExt string
	treeExt string
}

func NewSingleCellPathFactory(rootDir string) *SingleCellPathFactory {
	return &SingleCellPathFactory{rootDir: rootDir, fileExt: ".jkl", treeExt: ".tree"}
}

func (s *SingleCellPathFactory) FilePath(fid string) (path string) {
	return filepath.Join(s.FileDir(fid), s.fileName(fid))
}

func (s *SingleCellPathFactory) fileName(fid string) (name string) {
	var b strings.Builder
	// ignore length of string and nil error
	_, _ = b.WriteString(string(fid))
	_, _ = b.WriteString(s.fileExt)

	return b.String()
}

func (s *SingleCellPathFactory) FileDir(fid string) (dir string) {
	return filepath.Join(s.rootDir, "storage", string(fid))
}

func (s *SingleCellPathFactory) TreePath(fid string) (path string) {
	return filepath.Join(s.FileDir(fid), s.treeName(fid))
}

func (s *SingleCellPathFactory) treeName(fid string) (name string) {
	var b strings.Builder
	_, _ = b.WriteString(string(fid))
	_, _ = b.WriteString(s.treeExt)
	return b.String()
}

type MultiCellPathFactory struct {
    rootDir string
    fileExt string
    treeExt string
}

func NewMultiCellPathFactory(rootDir string) *MultiCellPathFactory {
    return &MultiCellPathFactory{rootDir: rootDir, fileExt: ".jkl", treeExt: ".tree"}
}

func (m *MultiCellPathFactory) PiecePath(fid string, index int) (path string) {
    return filepath.Join(m.rootDir, "storage", fmt.Sprintf("%d.jkl", index))
}

// Reads directory that stores fid, returns the last piece of fid.
// It only accounts for file names with <index>.jkl format where index is an int.
// It returns os.NotExists error when there are no files with such format.
func (m *MultiCellPathFactory) LastPiece(fid string) (int, error) {
    entries, err := os.ReadDir(m.FileDir(fid))
    if err != nil {
        return 0, err
    }
    // work backwards since the entries are in sorted order
    i := len(entries) - 1
    for ; i > 0; i++ {
        if entries[i].IsDir() {
            continue
        }
        ext := filepath.Ext(entries[i].Name())
        if ext != m.fileExt {
            continue
        }
        subStr, _ := strings.CutSuffix(entries[i].Name(), m.fileExt)
        index, err := strconv.ParseInt(subStr, 10, 0)
        if err != nil {
            continue
        }
        return int(index), nil
    }
    return 0, os.ErrNotExist
}

// returns the last piece of the file
func (m *MultiCellPathFactory) FilePath(fid string) (path string) {
	return filepath.Join(m.FileDir(fid), m.fileName(fid))
}

func (m *MultiCellPathFactory) FileDir(fid string) (dir string) {
	return filepath.Join(m.rootDir, "storage", string(fid))
}

func (m *MultiCellPathFactory) fileName(fid string) (name string) {
	var b strings.Builder
	// ignore length of string and nil error
	_, _ = b.WriteString(string(fid))
	_, _ = b.WriteString(m.fileExt)

	return b.String()
}

func (m *MultiCellPathFactory) TreePath(fid string) (path string) {
    return filepath.Join(m.rootDir, "storage", m.treeName(fid))
}

func (m *MultiCellPathFactory) TreeDir(fid string) (dir string) {
    return filepath.Join(m.rootDir, "storage")
}

func (m *MultiCellPathFactory) treeName(fid string) (name string) {
	var b strings.Builder
	_, _ = b.WriteString(string(fid))
	_, _ = b.WriteString(m.treeExt)
	return b.String()
}
