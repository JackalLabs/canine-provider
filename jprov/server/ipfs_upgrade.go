package server

import (
	"errors"
	"log/slog"
	"strconv"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

func (f *FileServer) PrepareIpfsUpgrade() error {
	err := f.PruneExpiredFiles()
	if err != nil {
		return err
	}

	return nil
}

func (f *FileServer) migrateToIpfs(activeDeal storageTypes.ActiveDeals) error {
	tree, err := f.archive.RetrieveTree(activeDeal.Fid)
	if err != nil {
		return err
	}

	file, err := f.archive.RetrieveFile(activeDeal.Fid)
	defer file.Close()
	if err != nil {
		return err
	}

	startBlock, err := strconv.ParseInt(activeDeal.Startblock, 10, 64)
	if err != nil {
		return errors.Join(errors.New("failed to parse startblock"), err)
	}

	err = f.ipfsArchive.WriteTreeToDisk(activeDeal.Merkle, activeDeal.Signee, startBlock, tree)
	if err != nil {
		return err
	}

	_, err = f.ipfsArchive.WriteFileToDisk(file, string(tree.Root()))
	return err
}

func (f *FileServer) IpfsUpgrade() error {
	activeDeals, err := f.QueryOnlyMyActiveDeals()
	if err != nil {
		return errors.Join(errors.New("failed to collect active deals"), err)
	}

	for _, a := range activeDeals {
		err = f.migrateToIpfs(a)
		if err != nil {
			f.logger.LogAttrs(
				f.cmd.Context(),
				slog.LevelError,
				"failed to migrate file to ipfs",
				slog.Any("error", err),
				activeDealLogAttr(a),
			)
		}
	}

	return nil
}

func activeDealLogAttr(activeDeal storageTypes.ActiveDeals) slog.Attr {
	return slog.Group(
		"activeDeal",
		slog.String("cid", activeDeal.Cid),
		slog.String("fid", activeDeal.Fid),
		slog.String("signee", activeDeal.Signee),
		slog.String("provider", activeDeal.Provider),
		slog.String("merkle", activeDeal.Merkle),
		slog.String("startBlock", activeDeal.Startblock),
	)
}
