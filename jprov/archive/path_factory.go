package archive

import (
	"path/filepath"
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
	return filepath.Join(s.rootDir, "storage", fid)
}

func (s *SingleCellPathFactory) TreePath(fid string) (path string) {
	return filepath.Join(s.FileDir(fid), s.treeName(fid))
}

func (s *SingleCellPathFactory) treeName(fid string) (name string) {
	var b strings.Builder
	_, _ = b.WriteString(fid)
	_, _ = b.WriteString(s.treeExt)
	return b.String()
}
