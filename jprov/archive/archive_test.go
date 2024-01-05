package archive_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
)

func NewFile(name string, t *testing.T) (f *os.File) {
	f, err := os.CreateTemp("", "_GO_"+name)
	if err != nil {
		t.Fatalf("TempFile %s: %s", name, err)
	}
	return
}

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
	if string(resData) != "orld\n\x00\x00\x00" {
		t.Errorf("GetPiece 1, 8: have %q, want %q", string(resData), "world")
	}
}
