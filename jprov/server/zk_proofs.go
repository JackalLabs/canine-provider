package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/jackalLabs/canine-chain/x/storage/zk"

	"github.com/wealdtech/go-merkletree/sha3"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"

	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"

	merkletree "github.com/wealdtech/go-merkletree"
)

var ccs, _ = zk.GetCircuit()

func CreateMerkleForZKProof(clientCtx client.Context, filename string, index int, ctx *utils.Context) (*zk.WrappedProof, []byte, string, error) {
	files := utils.GetStoragePathForPiece(clientCtx, filename, index)

	item, err := os.ReadFile(files) // read only the chunk we need
	if err != nil {
		ctx.Logger.Error("Error can't open file!")
		return nil, nil, "", err
	}

	rawTree, err := os.ReadFile(utils.GetStoragePathForTree(clientCtx, filename))
	if err != nil {
		ctx.Logger.Error("Error can't find tree!")
		return nil, nil, "", err
	}

	tree, err := merkletree.ImportMerkleTree(rawTree, sha3.New512()) // import the tree instead of creating the tree on the fly
	if err != nil {
		ctx.Logger.Error("Error can't import tree!")
		return nil, nil, "", err
	}

	wp, hashValue, err := zk.HashData(item, ccs)
	if err != nil {
		return wp, hashValue, "", err
	}

	h := sha256.New()
	_, err = io.WriteString(h, fmt.Sprintf("%d%x", index, hashValue))
	if err != nil {
		return wp, hashValue, "", err
	}
	ditem := h.Sum(nil)

	proof, err := tree.GenerateProof(ditem, 0)
	if err != nil {
		return wp, hashValue, "", err
	}

	jproof, err := json.Marshal(*proof)
	if err != nil {
		return wp, hashValue, "", err
	}

	verified, err := merkletree.VerifyProofUsing(ditem, false, proof, [][]byte{tree.Root()}, sha3.New512())
	if err != nil {
		ctx.Logger.Error(err.Error())
		return wp, hashValue, "", err
	}

	if !verified {
		ctx.Logger.Info("Cannot verify")
	}

	return wp, hashValue, string(jproof), nil
}

func postZKProof(clientCtx client.Context, cid string, block int64, db *leveldb.DB, q *queue.UploadQueue, ctx *utils.Context) error {
	data, err := db.Get(utils.MakeFileKey(cid), nil)
	if err != nil {
		return err
	}

	wp, hashValue, hashlist, err := CreateMerkleForZKProof(clientCtx, string(data), int(block), ctx)
	if err != nil {
		return err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return err
	}

	wpp, err := wp.Encode()
	if err != nil {
		ctx.Logger.Error(err.Error())
		return err
	}

	hash64 := base64.StdEncoding.EncodeToString(hashValue)

	msg := storagetypes.NewMsgPostZKProof(
		address,
		hash64,
		*wpp,
		hashlist,
		cid,
	)
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	u := types.Upload{
		Message:  msg,
		Err:      nil,
		Callback: &wg,
		Response: nil,
	}

	go func() {
		q.Append(&u)
		wg.Wait()

		if u.Err != nil {

			ctx.Logger.Error(fmt.Sprintf("Posting Error: %s", u.Err.Error()))
			return
		}

		if u.Response.Code != 0 {
			ctx.Logger.Error("Contract Response Error: %s", fmt.Errorf(u.Response.RawLog))
			return
		}
	}()

	return nil
}
