package archive_test

import (
    "testing"
    "os"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
)

func newFile(name string, t *testing.T) (f *os.File) {
	f, err := os.CreateTemp("", "_GO_"+name)
	if err != nil {
		t.Fatalf("TempFile %s: %s", name, err)
	}
	return
}

func TestGetPiece(t *testing.T) {
	file := newFile("TestGetPiece", t)
	defer os.Remove(file.Name())
	defer file.Close()

    archive := archive.NewSingleCellArchive("")

	const data = "hello, world\n"
	_, err := file.WriteString(data)
	if err != nil {
		t.Fatalf("WriteString: %s", err)
	}

    _, err = archive.WriteFileToDisk(file, file.Name())
    if err != nil {
        t.Fatalf("archive.WriteFileToDisk: %s", err)
    }

	resData, resErr := archive.GetPiece(file.Name(), 0, 5)
	if err != nil {
		t.Errorf("GetPiece 0, 5: %s", resErr)
	}
	if string(resData) != "hello" {
		t.Errorf("GetPiece 0, 5: have %q, want %q", string(resData), "hello")
	}

	resData, resErr = archive.GetPiece(file.Name(), 1, 5)
	if err != nil {
		t.Errorf("GetPiece 1, 5: %s", resErr)
	}
	if string(resData) != ", wor" {
		t.Errorf("GetPiece 1, 5: have %q, want %q", string(resData), ", wor")
	}

	// Test reading a block that starts almost at the end
	resData, resErr = archive.GetPiece(file.Name(), 1, 8)
	if err != nil {
		t.Errorf("GetPiece 1, 8: %s", resErr)
	}
	if string(resData) != "orld\n\x00\x00\x00" {
		t.Errorf("GetPiece 1, 8: have %q, want %q", string(resData), "world")
	}
}
