package archive

import (
	"errors"
	"os"
	"io"
	"path/filepath"
)

const FilePerm os.FileMode = 0o666

type Archive interface {
	// WriteToDisk creates a file at path and writes the contents read from data
	// It creates directory to write file at path
	// e.g. ~/dir/subdir/filename
	// Returns bytes written and nil error if write was successful
	// Returns non-nil error when it fails to create directory or create and write file
	WriteToDisk(data io.Reader, fid string) (written int64, err error)
	// GetPiece returns a piece of block at index of a file
	GetPiece(fid string, index, blockSize int64) (block []byte, err error)
	// RetrieveFile returns an io.ReaderCloser for the file data
	// The data must be closed after reading is done
	// Error is returned when such file does not exist
	RetrieveFile(fid string) (data io.ReadCloser, err error)
}

var _ Archive = &FileArchive{}

type FileArchive struct {
	rootDir string
	pathFactory pathFactory
}

func (f *FileArchive) WriteToDisk(data io.Reader, fid string) (written int64, err error) {
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

func (f *FileArchive) GetPiece(fid string, index, blockSize int64) (block []byte, err error) {
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

func (f *FileArchive) RetrieveFile(fid string) (data io.ReadCloser, err error) {
	data, err = os.Open(f.pathFactory.FilePath(fid))
	return
}
