package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"

	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"

	merkletree "github.com/wealdtech/go-merkletree"

	"github.com/spf13/cobra"
)

func CreateMerkleForProof(clientCtx client.Context, filename string, index int) (string, string, error) {
	files := utils.GetStoragePath(clientCtx, filename)

	var data [][]byte

	var item []byte

	for i := 0; i < len(files); i += 1 {
		f, err := os.ReadFile(filepath.Join(files, fmt.Sprintf("%d.jkl", i)))
		if err != nil {
			fmt.Printf("Error can't open file!\n")
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

	tree, err := merkletree.New(data)
	if err != nil {
		return "", "", err
	}

	h := sha256.New()
	_, err = io.WriteString(h, fmt.Sprintf("%d%x", index, item))
	if err != nil {
		return "", "", err
	}
	ditem := h.Sum(nil)

	proof, err := tree.GenerateProof(ditem)
	if err != nil {
		return "", "", err
	}

	jproof, err := json.Marshal(*proof)
	if err != nil {
		return "", "", err
	}

	e := hex.EncodeToString(tree.Root())

	k, err := hex.DecodeString(e)
	if err != nil {
		fmt.Println(err)
		return "", "", err

	}

	verified, err := merkletree.VerifyProof(ditem, proof, k)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	if !verified {
		fmt.Println("Cannot verify")
	}

	return fmt.Sprintf("%x", item), string(jproof), nil
}

func postProof(clientCtx client.Context, cid string, block string, db *leveldb.DB, queue *queue.UploadQueue) (*sdk.TxResponse, error) {
	dex, err := strconv.Atoi(block)
	if err != nil {
		return nil, err
	}

	data, err := db.Get(utils.MakeFileKey(cid), nil)
	if err != nil {
		return nil, err
	}

	item, hashlist, err := CreateMerkleForProof(clientCtx, string(data), dex)
	if err != nil {
		return nil, err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		fmt.Println(err)
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

	queue.Append(&u)
	wg.Wait()

	return u.Response, u.Err
}

func postProofs(cmd *cobra.Command, db *leveldb.DB, queue *queue.UploadQueue) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return
	}
	interval, err := cmd.Flags().GetUint16("interval")
	if err != nil {
		return
	}
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return
	}

	const maxMisses = 8

	for {

		iter := db.NewIterator(nil, nil)
		for iter.Next() {
			cid := string(iter.Key())
			value := string(iter.Value())

			if cid[:len(utils.FILE_KEY)] != utils.FILE_KEY {
				continue
			}

			cid = cid[len(utils.FILE_KEY):]

			if debug {
				fmt.Printf("filename: %s\n", value)
			}

			if debug {
				fmt.Printf("CID: %s\n", cid)
			}

			ver, verr := checkVerified(&clientCtx, cid)
			if verr != nil {
				// fmt.Println("Verification error")
				// fmt.Printf("ERROR: %v\n", verr)
				// fmt.Println(verr.Error())

				val, err := db.Get(utils.MakeDowntimeKey(cid), nil)
				newval := 0
				if err == nil {
					newval, err = strconv.Atoi(string(val))
					if err != nil {
						continue
					}
				}
				fmt.Printf("%s will be removed in %d cycles\n", value, maxMisses-newval)
				newval += 1

				if newval > maxMisses {
					os.RemoveAll(utils.GetStoragePath(clientCtx, value))
					err = db.Delete(utils.MakeFileKey(cid), nil)
					if err != nil {
						continue
					}
					err = db.Delete(utils.MakeDowntimeKey(cid), nil)
					if err != nil {
						continue
					}
					// err = db.Delete(makeUptimeKey(cid), nil)
					// if err != nil {
					// 	continue
					// }
					continue
				}

				err = db.Put(utils.MakeDowntimeKey(cid), []byte(fmt.Sprintf("%d", newval)), nil)
				if err != nil {
					continue
				}
				continue
			}

			if ver {
				if debug {
					fmt.Println("Skipping file as it's already verified.")
				}
				continue
			}

			block, berr := queryBlock(&clientCtx, string(cid))
			if berr != nil {
				fmt.Printf("Query Error: %v\n", berr)
				continue
			}

			res, err := postProof(clientCtx, cid, block, db, queue)
			if err != nil {
				fmt.Printf("Posting Error: %s\n", err.Error())
				continue
			}

			if res.Code != 0 {
				fmt.Printf("Contract Response Error: %s\n", fmt.Errorf(res.RawLog))
				continue
			}
		}
		iter.Release()
		err = iter.Error()
		if err != nil {
			fmt.Printf("Iterator Error: %s\n", err.Error())
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}
