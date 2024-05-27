package archive

import (
	"context"
	"errors"
	"fmt"
	"io"

	badger "github.com/dgraph-io/badger/v4"
	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/libp2p/go-libp2p/core/crypto"
	multiaddr "github.com/multiformats/go-multiaddr"

	merkletree "github.com/wealdtech/go-merkletree"

	bds "github.com/ipfs/go-ds-badger2"
)
import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type IpfsArchive struct {
	db   *badger.DB
	ipfs *ipfslite.Peer
}

func merkleTreeKey(merkle []byte, owner string, start int64) []byte {
	return []byte(fmt.Sprintf("tree/%x/%s/%d", merkle, owner, start))
}

func (i *IpfsArchive) WriteTreeToDisk(merkle string, owner string, start int64, tree *merkletree.MerkleTree) (err error) {
	k := merkleTreeKey([]byte(merkle), owner, start)
	v, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	err = i.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(k, v)
		if err != nil {
			return errors.Join(errors.New("failed to save tree"), err)
		}

		return nil
	})

	return nil
}

func fidKey(fid string) []byte {
	return []byte(fmt.Sprintf("cid/%x", []byte(fid)))
}

func (i *IpfsArchive) WriteFileToDisk(data io.Reader, fid string) (written int64, err error) {
	node, err := i.ipfs.AddFile(context.Background(), data, nil)
	if err != nil {
		return 0, err
	}

	wrote, err := node.Size()
	if err != nil {
		return 0, errors.Join(fmt.Errorf("failed to get bytes written"), err)
	}

	err = i.db.Update(func(txn *badger.Txn) error {
		return txn.Set(fidKey(fid), []byte(node.Cid().String()))
	})

	if err != nil {
		return int64(wrote), errors.Join(errors.New("failed to record fid to database"), err)
	}

	return int64(wrote), nil
}

func NewIpfsArchive(db *badger.DB, port int) (*IpfsArchive, error) {
	peer, err := newIpfsArchive(context.Background(), db, port)
	if err != nil {
		return nil, err
	}

	return &IpfsArchive{db: db, ipfs: peer}, nil
}

func newIpfsArchive(ctx context.Context, db *badger.DB, port int) (*ipfslite.Peer, error) {
	ds, err := bds.NewDatastoreFromDB(db)
	if err != nil {
		return nil, err
	}

	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		return nil, err
	}

	listen, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	h, dht, err := ipfslite.SetupLibp2p(
		ctx,
		priv,
		nil,
		[]multiaddr.Multiaddr{listen},
		ds,
		ipfslite.Libp2pOptionsExtra...,
	)
	if err != nil {
		return nil, err
	}

	lite, err := ipfslite.New(ctx, ds, nil, h, dht, nil)
	if err != nil {
		return nil, err
	}

	lite.Bootstrap(ipfslite.DefaultBootstrapPeers())

	return lite, nil
}
