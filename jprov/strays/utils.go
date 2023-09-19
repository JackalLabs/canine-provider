package strays

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
)

// NOTE: this is exact copy of server.WriteToDisk. It's copied here to avoid import cycle. This needs to be fixed!
const FilePerm os.FileMode = 0o666

func WriteToDisk(data io.Reader, closer io.Closer, dir, name string) (written int64, err error) {
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return
	}

	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE, FilePerm)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())

		if closer != nil {
			err = errors.Join(err, closer.Close())
		}
	}()

	written, err = io.Copy(file, data)
	if err != nil {
		err = fmt.Errorf("WriteToDisk: failed to write data to disk (wrote %d bytes)", written)
		return
	}

	return
}

func (h *LittleHand) DownloadFileFromURL(url string, fid string, cid string) (err error) {
	h.Logger.Info(fmt.Sprintf("Getting %s from %s", fid, url))

	cli := http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/download/%s", url, fid), nil)
	if err != nil {
		return
	}

	req.Header = http.Header{
		"User-Agent":                {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36"},
		"Upgrade-Insecure-Requests": {"1"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Encoding":           {"gzip, deflate, br"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Connection":                {"keep-alive"},
	}

	resp, err := cli.Do(req)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to find file on network")
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

    fileSize, err := h.Archive.WriteFileToDisk(resp.Body, fid)
	if err != nil {
		h.Logger.Error("saveFile: Write To Disk Error: ", err)
		return
	}

	blockSize, err := h.Cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		return
	}

    file, err := h.Archive.RetrieveFile(fid)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	// Create merkle and save to disk
	merkle, err := utils.CreateMerkleTree(blockSize, fileSize, file, file)
	if err != nil {
		return err
	}

	return h.Archive.WriteTreeToDisk(fid, merkle)
}
