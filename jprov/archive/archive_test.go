package archive_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
)

func TestGetPiece(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_, err := buf.WriteString("hello, world\n")
	if err != nil {
		t.Fatal(err)
	}

	archive := archive.NewSingleCellArchive("")

	fileName := "testfile"
	_, err = archive.WriteFileToDisk(buf, fileName)
	if err != nil {
		t.Fatalf("archive.WriteFileToDisk: %s", err)
	}
	defer func() {
		if err := os.RemoveAll("storage"); err != nil {
			t.Fatal(err)
			return
		}
	}()

	resData, resErr := archive.GetPiece(fileName, 0, 5)
	if err != nil {
		t.Errorf("GetPiece 0, 5: %s", resErr)
	}
	if string(resData) != "hello" {
		t.Errorf("GetPiece 0, 5: have %q, want %q", string(resData), "hello")
	}

	resData, resErr = archive.GetPiece(fileName, 1, 5)
	if err != nil {
		t.Errorf("GetPiece 1, 5: %s", resErr)
	}
	if string(resData) != ", wor" {
		t.Errorf("GetPiece 1, 5: have %q, want %q", string(resData), ", wor")
	}

	// Test reading a block that starts almost at the end
	resData, resErr = archive.GetPiece(fileName, 1, 8)
	if err != nil {
		t.Errorf("GetPiece 1, 8: %s", resErr)
	}
	if string(resData) != "orld\n" {
		t.Errorf("GetPiece 1, 8: have %q, want %q", string(resData), "orld\n")
	}
}

func prepareTestDir(rootDir string) (string, error) {
    tmpDir, err := os.MkdirTemp(rootDir, "")
    if err != nil {
        return "", err
    }
    err = os.Mkdir(filepath.Join(tmpDir, "storage"), 0755)
    if err != nil {
        err = errors.Join(err, os.RemoveAll(tmpDir))
        return "", err
    }

    return tmpDir, nil
}

func TestHybridCellArchiveGetLegacyPiece(t *testing.T) {
    tmpRootDir, err := prepareTestDir(".")
    if err != nil {
        t.Errorf("failed to create temporary directory for testing: %v", err)
    }
    defer func() {
        err = os.RemoveAll(tmpRootDir)
        if err != nil {
            t.Errorf("failed to delete testing directory: %v", err)
        }
    }()

    storageDir := filepath.Join(tmpRootDir, "storage")

    fid0Dir := filepath.Join(storageDir, "fid0")
    err = os.Mkdir(fid0Dir, 0755)
    if err != nil {
        t.Errorf("failed to make directory for fid0: %v", err)
    }

    zeroDotJkl, err := os.Create(filepath.Join(fid0Dir, "0.jkl"))
    if err != nil {
        t.Errorf("failed to create fid0 0.jkl file: %v", err)
    }
    defer func() {
        err = zeroDotJkl.Close()
        if err != nil {
            t.Errorf("failed to close fid0 0.jkl: %v", err)
        }
    }()

    contents := []byte("hello world!\n")
    _, err = zeroDotJkl.Write(contents)
    if err != nil {
        t.Errorf("failed to write test contents at fid0 0.jkl: %v", err)
    }
    
    // fid0.jkl might exist in the same dir if migrations is also happening
    fid0DotJkl, err := os.Create(filepath.Join(fid0Dir, "fid0.jkl"))
    if err != nil {
        t.Errorf("failed to create fid0 0.jkl file: %v", err)
    }
    defer func() {
        err = fid0DotJkl.Close()
        if err != nil {
            t.Errorf("failed to close fid0 0.jkl: %v", err)
        }
    }()

    _, err = fid0DotJkl.Write(contents)
    if err != nil {
        t.Errorf("failed to write test contents at fid0 0.jkl: %v", err)
    }

    hybrid := archive.NewHybridCellArchive(tmpRootDir)

    data, err := hybrid.GetPiece("fid0", 0, int64(len(contents)))
    if err != nil {
        t.Errorf("GetPiece fid0, 0, %d: unexpected error %v", len(contents), err)
    }

    if string(data) != string(contents) {
        t.Errorf(
            "GetPiece fid0, 0, %d: have %q, want %q", 
            len(contents), 
            string(data), 
            string(contents),
        )
    }
}

func TestHybridCellArchiveGetSingleCellPiece(t *testing.T) {
    tmpRootDir, err := prepareTestDir(".")
    if err != nil {
        t.Errorf("failed to create temporary directory for testing: %v", err)
    }
    defer func() {
        err = os.RemoveAll(tmpRootDir)
        if err != nil {
            t.Errorf("failed to delete testing directory: %v", err)
        }
    }()

    storageDir := filepath.Join(tmpRootDir, "storage")

    fid0Dir := filepath.Join(storageDir, "fid0")
    err = os.Mkdir(fid0Dir, 0755)
    if err != nil {
        t.Errorf("failed to make directory for fid0: %v", err)
    }

    fid0, err := os.Create(filepath.Join(fid0Dir, "fid0.jkl"))
    if err != nil {
        t.Errorf("failed to create fid0 0.jkl file: %v", err)
    }
    defer func() {
        err = fid0.Close()
        if err != nil {
            t.Errorf("failed to close fid0 0.jkl: %v", err)
        }
    }()

    contents := []byte("hello world!\n")
    _, err = fid0.Write(contents)
    if err != nil {
        t.Errorf("failed to write test contents at fid0 0.jkl: %v", err)
    }
    
    hybrid := archive.NewHybridCellArchive(tmpRootDir)

    data, err := hybrid.GetPiece("fid0", 0, int64(len(contents)))
    if err != nil {
        t.Errorf("GetPiece fid0, 0, %d: unexpected error %v", len(contents), err)
    }

    if string(data) != string(contents) {
        t.Errorf(
            "GetPiece fid0, 0, %d: have %q, want %q", 
            len(contents), 
            string(data), 
            string(contents),
        )
    }
}
