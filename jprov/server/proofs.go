package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

func CreateMerkleForProof(clientCtx client.Context, filename string, index int, ctx *utils.Context) (string, string, error) {
	files := utils.GetStoragePathForPiece(clientCtx, filename, index)

	item, err := os.ReadFile(files) // read only the chunk we need
	if err != nil {
		ctx.Logger.Error("Error can't open file!")
		return "", "", err
	}

	rawTree, err := os.ReadFile(utils.GetStoragePathForTree(clientCtx, filename))
	if err != nil {
		ctx.Logger.Error("Error can't find tree!")
		return "", "", err
	}

	tree, err := merkletree.ImportMerkleTree(rawTree, sha3.New512()) // import the tree instead of creating the tree on the fly
	if err != nil {
		ctx.Logger.Error("Error can't import tree!")
		return "", "", err
	}

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

func postProof(clientCtx client.Context, cid string, block string, db *leveldb.DB, q *queue.UploadQueue, ctx *utils.Context) error {
	dex, ok := sdk.NewIntFromString(block)
	ctx.Logger.Debug(fmt.Sprintf("BlockToProve: %s", block))
	if !ok {
		return fmt.Errorf("cannot parse block number")
	}

	data, err := db.Get(utils.MakeFileKey(cid), nil)
	if err != nil {
		return err
	}

	item, hashlist, err := CreateMerkleForProof(clientCtx, string(data), int(dex.Int64()), ctx)
	if err != nil {
		return err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return err
	}

	msg := storagetypes.NewMsgPostproof(
		address,
		item,
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

func postProofs(cmd *cobra.Command, db *leveldb.DB, q *queue.UploadQueue, ctx *utils.Context) {
	intervalFromCMD, err := cmd.Flags().GetUint16(types.FlagInterval)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}

	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}

	maxMisses, err := cmd.Flags().GetInt(types.FlagMaxMisses)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		interval := intervalFromCMD

		// if interval < 1800 { // If the provider picked an interval that's less than 30 minutes, we generate a random interval for them anyways

		// 	r := rand.New(rand.NewSource(time.Now().UnixNano()))
		// 	interval = uint16(r.Intn(901) + 900) // Generate interval between 15-30 minutes

		// }
		ctx.Logger.Debug(fmt.Sprintf("The interval between proofs is now %d", interval))
		start := time.Now()

		iter := db.NewIterator(nil, nil)

		for iter.Next() {
			cid := string(iter.Key())
			value := string(iter.Value())

			if cid[:len(utils.FileKey)] != utils.FileKey {
				continue
			}

			cid = cid[len(utils.FileKey):]

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

					duplicate := false
					iter := db.NewIterator(nil, nil)
					for iter.Next() {
						c := string(iter.Key())
						v := string(iter.Value())

						if c[:len(utils.FileKey)] != utils.FileKey {
							continue
						}

						c = c[len(utils.FileKey):]

						if c != cid && v == value {
							ctx.Logger.Info(fmt.Sprintf("%s != %s but it is also %s, so we must keep the file on disk.", c, cid, v))
							duplicate = true
							break
						}
					}
					ctx.Logger.Info(fmt.Sprintf("%s is being removed", cid))

					if !duplicate {
						ctx.Logger.Info("And we are removing the file on disk.")

						err := os.RemoveAll(utils.GetStoragePath(clientCtx, value))
						if err != nil {
							ctx.Logger.Error(err.Error())
						}

						err = os.Remove(utils.GetStoragePathForTree(clientCtx, value))
						if err != nil {
							ctx.Logger.Error(err.Error())
							continue
						}
					}
					err = db.Delete(utils.MakeFileKey(cid), nil)
					if err != nil {
						ctx.Logger.Error(err.Error())
						continue
					}

					err = db.Delete(utils.MakeDowntimeKey(cid), nil)
					if err != nil {
						ctx.Logger.Error(err.Error())
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

			block, berr := queryBlock(&clientCtx, cid)
			if berr != nil {
				ctx.Logger.Error(fmt.Sprintf("Query Error: %v", berr))
				continue
			}

			err = postProof(clientCtx, cid, block, db, q, ctx)
			if err != nil {
				ctx.Logger.Error(fmt.Sprintf("Posting Proof Error: %v", err))
				continue
			}
			// time.Sleep(time.Second) // remove the sleep after posting proof

		}

		iter.Release()
		err = iter.Error()
		if err != nil {
			ctx.Logger.Error("Iterator Error: %s", err.Error())
		}

		end := time.Since(start)
		if end.Seconds() > 120 {
			ctx.Logger.Error(fmt.Sprintf("proof took %d", end.Nanoseconds()))
		}

		tm := time.Duration(interval) * time.Second

		if tm.Nanoseconds()-end.Nanoseconds() > 0 {
			time.Sleep(time.Duration(interval) * time.Second)
		}

	}
}
