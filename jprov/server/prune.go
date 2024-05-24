package server

import (
	"errors"
	"fmt"
	"os"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
)

func (f *FileServer) allFilesAtStorage() ([]string, error) {
	fids := make([]string, 0)
	dirs, err := os.ReadDir(utils.GetStorageRootDir(f.serverCtx.cosmosCtx.HomeDir))
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			fids = append(fids, dir.Name())
		}
	}

	return fids, err
}

func findOldMerkleTree(homedir, fid string) (bool, error) {
	oldTreePath := utils.GetOldTreePath(homedir, fid)

	_, err := os.Stat(oldTreePath)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	// we should never reach here because at this point file exist and not exist
	// manual investigation required at this point
	return false, fmt.Errorf("Error: FindMigratedFile: file exist and not exist at the same time: %s", err.Error())
}

func (f *FileServer) purge(fid string) error {
	exists, err := findOldMerkleTree(f.serverCtx.cosmosCtx.HomeDir, fid)
	if err != nil {
		return err
	}

	if exists {
		if err := os.Remove(utils.GetOldTreePath(f.serverCtx.cosmosCtx.HomeDir, fid)); err != nil {
			return err
		}
	}

	return f.archive.Delete(fid)
}

func (f *FileServer) PruneExpiredFiles() error {
	fids, err := f.allFilesAtStorage()
	if err != nil {
		return err
	}

	count := 0
	for _, fid := range fids {
		_, err := f.archivedb.GetContracts(fid)
		if errors.Is(err, archive.ErrFidNotFound) {
			err := f.purge(fid)
			if err != nil {
				return err
			}
			count++
		} else if err != nil {
			return err
		}
	}

	fmt.Printf("pruned %d out of %d files", count, len(fids))
	return err
}
