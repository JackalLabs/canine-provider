package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/wealdtech/go-merkletree/sha3"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/client"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"

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
	hashBuilder.WriteString(strconv.FormatInt(index, 10))
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
	fid, err := f.archivedb.GetFid(cid)
	if err != nil {
		return err
	}

	item, hashlist, err := f.CreateMerkleForProof(fid, blockSize, block)
	if err != nil {
		return err
	}

	fmt.Printf("Requesting attestion for: %s\n", cid)

	err = requestAttestation(f.serverCtx.cosmosCtx, cid, hashlist, item, f.queue) // request attestation, if we get it, skip all the posting
	if err == nil {
		fmt.Println("successfully got attestation.")
		return nil
	}

	msg := storageTypes.NewMsgPostproof(
		f.serverCtx.address,
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

	f.queue.Append(&u)
	wg.Wait()

	if u.Err != nil {
		f.logger.Error(fmt.Sprintf("Posting Error: %s", u.Err.Error()))
		return nil
	}

	if u.Response.Code != 0 {
		f.logger.Error(fmt.Errorf("contract Response error: %s", u.Response.RawLog).Error())
		return nil
	}

	return nil
}

func (f *FileServer) Purge(cid string) error {
	fid, err := f.archivedb.GetFid(cid)
	if err != nil {
		return err
	}

	err = f.downtimedb.Delete(cid)
	if err != nil {
		return err
	}

	purge, err := f.archivedb.DeleteContract(cid)
	if err != nil {
		return err
	}

	if purge {
		err := f.archive.Delete(fid)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FileServer) CleanExpired() error {
	maxMisses, err := f.cmd.Flags().GetInt(types.FlagMaxMisses)
	if err != nil {
		f.logger.Error(err.Error())
		return err
	}

	iter := f.downtimedb.NewIterator()
	defer iter.Release()

	for iter.Next() {
		cid := string(iter.Key())
		downtime, err := archive.ByteToBlock(iter.Value())
		if err != nil {
			return err
		}

		if downtime > int64(maxMisses) {
			err := f.Purge(cid)
			if err != nil {
				return err
			}
			f.logger.Info(fmt.Sprintf("Purged CID: %s", string(cid)))
		} else {
			f.logger.Info(fmt.Sprintf("%s will be removed in %d cycles",
				string(cid), int64(maxMisses)-downtime))
		}
	}

	return nil
}

func (f *FileServer) IncrementDowntime(cid string) error {
	downtime, err := f.downtimedb.Get(cid)
	if err != nil && !errors.Is(err, archive.ErrContractNotFound) {
		return err
	}

	err = f.downtimedb.Set(cid, downtime+1)

	return err
}

func (f *FileServer) DeleteDowntime(cid string) error {
	_, err := f.downtimedb.Get(cid)
	if errors.Is(err, archive.ErrContractNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return f.downtimedb.Delete(cid)
}

func (f *FileServer) ContractState(cid string) string {
	return f.QueryContractState(cid)
}

func (f *FileServer) Prove(deal storageTypes.LegacyActiveDeals) error {
	dex, ok := sdk.NewIntFromString(deal.Blocktoprove)
	f.logger.Debug(fmt.Sprintf("BlockToProve: %s", deal.Blocktoprove))
	if !ok {
		return fmt.Errorf("failed to parse block number: %s", deal.Blocktoprove)
	}

	return f.postProof(deal.Cid, f.blockSize, dex.Int64())
}

func (f *FileServer) handleContracts() error {
	iter := f.archivedb.NewIterator()
	defer iter.Release()

	for iter.Next() {
		cid := string(iter.Key())
		fid := string(iter.Value())
		if strings.HasPrefix(cid, "jklf") { // skip cid reference
			continue
		}

		f.logger.Info(fmt.Sprintf("CID: %s FID: %s", cid, fid))
		resp, respErr := f.QueryActiveDeal(cid)

		switch state, err := types.ContractState(resp, respErr); state {
		case types.Verified:
			err := f.DeleteDowntime(cid)
			if err != nil {
				f.logger.Error(fmt.Sprintf("error when unmarking downtime cid: %s: %v", cid, err))
			}
			continue
		case types.NotFound:
			err := f.IncrementDowntime(cid)
			if err != nil {
				return err
			}
		case types.NotVerified:
			err := f.DeleteDowntime(cid)
			if err != nil {
				f.logger.Error(fmt.Sprintf("error when unmarking downtime cid: %s: %v", cid, err))
			}

			err = f.Prove(resp.ActiveDeals)
			if err != nil {
				f.logger.Error(fmt.Sprintf("failed to prove: %s: %v", cid, err))
			}
		case types.Error:
			f.logger.Error(fmt.Sprintf("query error: %v", err))
		default:
			return fmt.Errorf("unkown state: %v %v", state, err)
		}
	}
	return nil
}

func (f *FileServer) startShift() error {
	err := f.CleanExpired()
	if err != nil {
		return err
	}

	return f.handleContracts()
}

func (f *FileServer) StartProofServer(interval uint16) {
	// catch interrupt or termination sig and stop proving
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			fmt.Println("shutting down proof server")
			return
		default:
			start := time.Now()
			err := f.startShift()
			if err != nil {
				f.logger.Error(err.Error())
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
}
