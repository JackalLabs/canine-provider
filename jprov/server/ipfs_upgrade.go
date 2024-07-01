package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/v2/sha3"

	merkletree2 "github.com/wealdtech/go-merkletree/v2"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

func (f *FileServer) PrepareIpfsUpgrade() error {
	err := f.PruneExpiredFiles()
	if err != nil {
		return err
	}

	return nil
}

func (f *FileServer) migrateToIpfs(activeDeal storageTypes.LegacyActiveDeals) error {
	tree, err := f.archive.RetrieveTree(activeDeal.Fid)
	if err != nil {
		return err
	}
	fmt.Printf("Migrating %x...\n", tree.Root())

	file, err := f.archive.RetrieveFile(activeDeal.Fid)
	// nolint:all
	defer file.Close()
	if err != nil {
		return err
	}

	startBlock, err := strconv.ParseInt(activeDeal.Startblock, 10, 64)
	if err != nil {
		return errors.Join(errors.New("failed to parse startblock"), err)
	}

	merkle := make([]byte, hex.DecodedLen(len([]byte(activeDeal.Merkle))))
	_, err = hex.Decode(merkle, []byte(activeDeal.Merkle))
	if err != nil {
		return fmt.Errorf("failed to decode merkle | %w", err)
	}
	e, err := tree.Export()
	if err != nil {
		return fmt.Errorf("failed to temp export tree | %w", err)
	}
	var me merkletree.Export
	err = json.Unmarshal(e, &me)
	if err != nil {
		return fmt.Errorf("failed to temp re-import tree | %w", err)
	}

	t, err := merkletree2.NewTree(
		merkletree2.WithData(me.Data),
		merkletree2.WithHashType(sha3.New512()),
		merkletree2.WithSalt(me.Salt),
	)
	if err != nil {
		return err
	}
	err = f.ipfsArchive.WriteTreeToDisk(merkle, activeDeal.Signee, startBlock, t)
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

	return f.ipfsArchive.Stop()
}

func activeDealLogAttr(activeDeal storageTypes.LegacyActiveDeals) slog.Attr {
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
