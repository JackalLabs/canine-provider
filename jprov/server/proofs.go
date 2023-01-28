package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/wealdtech/go-merkletree/sha3"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"

	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"

	merkletree "github.com/wealdtech/go-merkletree"

	"github.com/spf13/cobra"
)

func CreateMerkleForProof(clientCtx client.Context, filename string, index int64, ctx *utils.Context) (string, string, error) {
	files := utils.GetStoragePath(clientCtx, filename)

	f, err := os.Open(files)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return "", "", err
	}

	fileInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		ctx.Logger.Error(err.Error())

		return "", "", err
	}

	var data [][]byte

	var item []byte

	var i int64
	for i = 0; i < int64(len(fileInfo)); i++ {

		f, err := os.ReadFile(filepath.Join(files, fmt.Sprintf("%d.jkl", i)))
		if err != nil {
			ctx.Logger.Error("Error can't open file!")
			return "", "", err
		}

		if i == index {
			item = f
		}

		h := sha256.New()
		_, err = io.WriteString(h, fmt.Sprintf("%d%x", i, f))
		if err != nil {
			return "", "", err
		}
		hashName := h.Sum(nil)

		data = append(data, hashName)
	}

	tree, err := merkletree.NewUsing(data, sha3.New512(), false)
	if err != nil {
		return "", "", err
	}

	ctx.Logger.Info(fmt.Sprintf("Merkle Root: %x", tree.Root()))

	h := sha256.New()
	_, err = io.WriteString(h, fmt.Sprintf("%d%x", index, item))
	if err != nil {
		return "", "", err
	}
	ditem := h.Sum(nil)

	proof, err := tree.GenerateProof(ditem, 0)
	if err != nil {
		return "", "", err
	}

	jproof, err := json.Marshal(*proof)
	if err != nil {
		return "", "", err
	}

	verified, err := merkletree.VerifyProofUsing(ditem, false, proof, [][]byte{tree.Root()}, sha3.New512())
	if err != nil {
		ctx.Logger.Error(err.Error())
		return "", "", err
	}

	if !verified {
		ctx.Logger.Info("Cannot verify")
	}

	return fmt.Sprintf("%x", item), string(jproof), nil
}

func postProof(clientCtx client.Context, cid string, block string, db *leveldb.DB, q *queue.UploadQueue, ctx *utils.Context) (*sdk.TxResponse, error) {
	dex, ok := sdk.NewIntFromString(block)
	ctx.Logger.Debug(fmt.Sprintf("BlockToProve: %s", block))
	if !ok {
		return nil, fmt.Errorf("cannot parse block number")
	}

	data, err := db.Get(utils.MakeFileKey(cid), nil)
	if err != nil {
		return nil, err
	}

	item, hashlist, err := CreateMerkleForProof(clientCtx, string(data), dex.Int64(), ctx)
	if err != nil {
		return nil, err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return nil, err
	}

	msg := storagetypes.NewMsgPostproof(
		address,
		item,
		hashlist,
		cid,
	)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	u := types.Upload{
		Message:  msg,
		Err:      nil,
		Callback: &wg,
		Response: nil,
	}

	q.Append(&u)
	wg.Wait()

	return u.Response, u.Err
}

func postProofs(cmd *cobra.Command, db *leveldb.DB, q *queue.UploadQueue, ctx *utils.Context) {
	interval, err := cmd.Flags().GetUint16("interval")
	if err != nil {
		return
	}

	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return
	}

	const maxMisses = 8

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {

		iter := db.NewIterator(nil, nil)
		for iter.Next() {
			cid := string(iter.Key())
			value := string(iter.Value())

			if cid[:len(utils.FILE_KEY)] != utils.FILE_KEY {
				continue
			}

			cid = cid[len(utils.FILE_KEY):]

			ctx.Logger.Debug(fmt.Sprintf("filename: %s", value))

			ctx.Logger.Debug(fmt.Sprintf("CID: %s", cid))

			ver, verr := checkVerified(&clientCtx, cid, address)
			if verr != nil {
				ctx.Logger.Error(verr.Error())
				rr := strings.Contains(verr.Error(), "key not found")
				ny := strings.Contains(verr.Error(), ErrNotYours)
				if !rr && !ny {
					continue
				}
				val, err := db.Get(utils.MakeDowntimeKey(cid), nil)
				newval := 0
				if err == nil {
					newval, err = strconv.Atoi(string(val))
					if err != nil {
						continue
					}
				}

				newval += 1

				if newval > maxMisses {
					ctx.Logger.Info(fmt.Sprintf("%s is being removed", value))
					os.RemoveAll(utils.GetStoragePath(clientCtx, value))
					err = db.Delete(utils.MakeFileKey(cid), nil)
					if err != nil {
						continue
					}
					err = db.Delete(utils.MakeDowntimeKey(cid), nil)
					if err != nil {
						continue
					}
					continue
				}

				ctx.Logger.Info(fmt.Sprintf("%s will be removed in %d cycles", value, (maxMisses+1)-newval))

				err = db.Put(utils.MakeDowntimeKey(cid), []byte(fmt.Sprintf("%d", newval)), nil)
				if err != nil {
					continue
				}
				continue
			}

			val, err := db.Get(utils.MakeDowntimeKey(cid), nil)
			newval := 0
			if err == nil {
				newval, err = strconv.Atoi(string(val))
				if err != nil {
					continue
				}
			}

			newval -= 1 // lower the downtime counter to only account for consecutive misses.
			if newval < 0 {
				newval = 0
			}

			err = db.Put(utils.MakeDowntimeKey(cid), []byte(fmt.Sprintf("%d", newval)), nil)
			if err != nil {
				continue
			}

			if ver {
				ctx.Logger.Debug("Skipping file as it's already verified.")
				continue
			}

			block, berr := queryBlock(&clientCtx, string(cid))
			if berr != nil {
				ctx.Logger.Error("Query Error: %v", berr)
				continue
			}

			res, err := postProof(clientCtx, cid, block, db, q, ctx)
			if err != nil {
				ctx.Logger.Error(fmt.Sprintf("Posting Error: %s", err.Error()))
				continue
			}

			if res.Code != 0 {
				ctx.Logger.Error("Contract Response Error: %s", fmt.Errorf(res.RawLog))
				continue
			}
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			ctx.Logger.Error("Iterator Error: %s", err.Error())
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}
