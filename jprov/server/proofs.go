package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
    "strconv"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/wealdtech/go-merkletree/sha3"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/client"

	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"

	merkletree "github.com/wealdtech/go-merkletree"
)

func GetMerkleTree(ctx client.Context, filename string) (*merkletree.MerkleTree, error) {
	rawTree, err := os.ReadFile(utils.GetStoragePathForTree(ctx.HomeDir, filename))
	if err != nil {
		return &merkletree.MerkleTree{}, fmt.Errorf("unable to find merkle tree for: %s", filename)
	}

	return merkletree.ImportMerkleTree(rawTree, sha3.New512())
}

func GenerateMerkleProof(tree merkletree.MerkleTree, index, blockSize int64, item []byte) (valid bool, proof *merkletree.Proof, err error) {
	h := sha256.New()

    var hashBuilder strings.Builder
    hashBuilder.WriteString(strconv.FormatInt(index/blockSize, 10))
    hashBuilder.WriteString(hex.EncodeToString(item))
	_, err = io.WriteString(h, hashBuilder.String())
	if err != nil {
		return
	}

	proof, err = tree.GenerateProof(h.Sum(nil), 0)
	if err != nil {
		return
	}

	valid, err = merkletree.VerifyProofUsing(h.Sum(nil), false, proof, [][]byte{tree.Root()}, sha3.New512())
	return
}

func (f *FileServer) CreateMerkleForProof(filename string, blockSize, index int64) (string, string, error) {
	data, err := f.archive.GetPiece(filename, index, blockSize)
	if err != nil {
		return "", "", err
	}

	mTree, err := f.archive.RetrieveTree(filename)
	if err != nil {
		return "", "", err
	}

	verified, proof, err := GenerateMerkleProof(*mTree, index, blockSize, data)
	if err != nil {
		f.logger.Error(err.Error())
		return "", "", err
	}

	jproof, err := json.Marshal(*proof)
	if err != nil {
		return "", "", err
	}

	if !verified {
		f.logger.Info("unable to generate valid proof")
	}

	return fmt.Sprintf("%x", data), string(jproof), nil
}

func requestAttestation(clientCtx client.Context, cid string, hashList string, item string, q *queue.UploadQueue) error {
	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		return err
	}

	msg := storageTypes.NewMsgRequestAttestationForm(
		address,
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

	q.Append(&u)
	wg.Wait()

	if u.Err != nil {
		fmt.Println(u.Err)
		return u.Err
	}

	if u.Response.Code != 0 {
		return fmt.Errorf(u.Response.RawLog)
	}

	var res storageTypes.MsgRequestAttestationFormResponse

	data, err := hex.DecodeString(u.Response.Data)
	if err != nil {
		fmt.Println(err)
		return err
	}

	var txMsgData sdk.TxMsgData

	err = clientCtx.Codec.Unmarshal(data, &txMsgData)
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, data := range txMsgData.Data {
		if data.GetMsgType() == "/canine_chain.storage.MsgRequestAttestationForm" {
			err := res.Unmarshal(data.Data)
			if err != nil {
				fmt.Println(err)
				return err
			}
			if res.Cid == cid {
				break
			}
		}
	}

	_ = clientCtx.PrintProto(&res)

	if !res.Success {
		fmt.Println("request form failed")
		fmt.Println(res.Error)
		return fmt.Errorf("failed to get attestations")
	}

	providerList := res.Providers
	var pwg sync.WaitGroup

	count := 0 // keep track of how many successful requests we've made

	for _, provider := range providerList { // request attestation from all providers, and wait until they all respond
		pwg.Add(1)

		prov := provider
		go func() {
			defer pwg.Done() // notify group that I have completed at the end of this function lifetime

			queryClient := storageTypes.NewQueryClient(clientCtx)

			provReq := &storageTypes.QueryProviderRequest{
				Address: prov,
			}

			providerDetails, err := queryClient.Providers(context.Background(), provReq)
			if err != nil {
				return
			}

			p := providerDetails.Providers
			providerAddress := p.Ip // get the providers IP address from chain at runtime

			path, err := url.JoinPath(providerAddress, "attest")
			if err != nil {
				return
			}

			attestRequest := types.AttestRequest{
				Cid:      cid,
				HashList: hashList,
				Item:     item,
			}

			data, err := json.Marshal(attestRequest)
			if err != nil {
				return
			}

			buf := bytes.NewBuffer(data)

			res, err := http.Post(path, "application/json", buf)
			if err != nil {
				return
			}

			if res.StatusCode == 200 {
				count += 1
			}
		}()

	}

	pwg.Wait()

	if count < 3 { // NOTE: this value can change in chain params
		fmt.Println("failed to get enough attestations...")
		return fmt.Errorf("failed to get attestations")
	}

	return nil
}

func (f *FileServer) postProof(cid string, blockSize, block int64) error {
    fid, err := f.archivedb.GetFid(string(cid))
    if err != nil {
        return err
    }

	item, hashlist, err := f.CreateMerkleForProof(string(fid), blockSize, block)
	if err != nil {
		return err
	}

	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		f.logger.Error(err.Error())
		return err
	}

	fmt.Printf("Requesting attestion for: %s\n", cid)

	err = requestAttestation(f.cosmosCtx, cid, hashlist, item, f.queue) // request attestation, if we get it, skip all the posting
	if err == nil {
		fmt.Println("successfully got attestation.")
		return nil
	}

	msg := storageTypes.NewMsgPostproof(
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
		f.queue.Append(&u)
		wg.Wait()

		if u.Err != nil {

			f.logger.Error(fmt.Sprintf("Posting Error: %s", u.Err.Error()))
			return
		}

		if u.Response.Code != 0 {
			f.logger.Error("Contract Response Error: %s", fmt.Errorf(u.Response.RawLog))
			return
		}
	}()

	return nil
}

func (f *FileServer) postProofs(interval uint16) {
	maxMisses, err := f.cmd.Flags().GetInt(types.FlagMaxMisses)
	if err != nil {
		f.logger.Error(err.Error())
		return
	}

	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
	// 1. Every interval, check contracts that needs proofs verified
		if interval == 0 { // If the provider picked an interval that's less than 30 minutes, we generate a random interval for them anyways

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			interval = uint16(r.Intn(3601) + 60) // Generate interval between 1-60 minutes

		}
		f.logger.Debug(fmt.Sprintf("The interval between proofs is now %d", interval))
		start := time.Now()

		iter := f.archivedb.NewIterator()

		for iter.Next() {
			// get next cid / fid
			cid := string(iter.Key())
			fid := string(iter.Value())

			f.logger.Debug(fmt.Sprintf("filename: %s", string(fid)))

			f.logger.Debug(fmt.Sprintf("CID: %s", cid))

			ver, verr := checkVerified(&f.cosmosCtx, string(cid), address)
			if verr != nil {
				f.logger.Error(verr.Error())
				rr := strings.Contains(verr.Error(), "key not found")
				ny := strings.Contains(verr.Error(), ErrNotYours)
				if !rr && !ny {
					continue
				}
                downtime, err := f.downtimedb.Get(string(cid))
                if err != nil {
                    f.logger.Error(err.Error())
                    continue
                }

				if downtime > int64(maxMisses) {
                    purge, err := f.archivedb.DeleteContract(string(cid))
                    if err != nil {
						f.logger.Error(err.Error())
                    } 
                    f.archive.Delete(fid)

					f.logger.Info(fmt.Sprintf("%s is being removed", cid))

                    if purge {
                        f.logger.Info("And we are removing the file on disk.")
                        f.archive.Delete(fid)
                    }

                    err = f.downtimedb.Delete(cid)
					if err != nil {
						f.logger.Error(err.Error())
						continue
					}
					continue
				}
                downtime += 1

				f.logger.Info(fmt.Sprintf("%s will be removed in %d cycles", string(fid), int64(maxMisses)-downtime))

                err = f.downtimedb.Set(cid, downtime)
				if err != nil {
                    f.logger.Error(err.Error())
				}
				continue
			}

            downtime, err := f.downtimedb.Get(cid)
            if err != nil {
                f.logger.Error(err.Error())
            }


			if downtime > 0 {
                downtime -= 1 // lower the downtime counter to only account for consecutive misses.
			}

            err = f.downtimedb.Set(cid, downtime)
			if err != nil {
                f.logger.Error(err.Error())
				continue
			}

			if ver {
				f.logger.Debug("Skipping file as it's already verified.")
				continue
			}

			block, berr := queryBlock(&f.cosmosCtx, string(cid))
			if berr != nil {
				f.logger.Error(fmt.Sprintf("Query Error: %v", berr))
				continue
			}

			dex, ok := sdk.NewIntFromString(block)
			f.logger.Debug(fmt.Sprintf("BlockToProve: %s", block))
			if !ok {
				f.logger.Error("cannot parse block number")
				continue
			}

			err = f.postProof(string(cid), f.blockSize, dex.Int64())
			if err != nil {
				f.logger.Error(fmt.Sprintf("Posting Proof Error: %v", err))
				continue
			}
			sleep, err := f.cmd.Flags().GetInt64(types.FlagSleep)
			if err != nil {
				f.logger.Error(err.Error())
				continue
			}
			time.Sleep(time.Duration(sleep) * time.Millisecond)

		}

		iter.Release()
		err = iter.Error()
		if err != nil {
			f.logger.Error("Iterator Error: %s", err.Error())
		}

		end := time.Since(start)
		if end.Seconds() > 120 {
			f.logger.Error(fmt.Sprintf("proof took %d", end.Nanoseconds()))
		}

		tm := time.Duration(interval) * time.Second

		if tm.Nanoseconds()-end.Nanoseconds() > 0 {
			time.Sleep(time.Duration(interval) * time.Second)
		}

	}
}
