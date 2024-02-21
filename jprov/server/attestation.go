package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	storageKeeper "github.com/jackalLabs/canine-chain/x/storage/keeper"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

func verifyAttest(deal storageTypes.ActiveDeals, attest types.AttestRequest) (verified bool, err error) {
	merkle := deal.Merkle
	block := deal.Blocktoprove
	blockNum, err := strconv.ParseInt(block, 10, 64)
	if err != nil {
		return false, err
	}

	verified = storageKeeper.VerifyDeal(merkle, attest.HashList, blockNum, attest.Item)

	return
}

func addMsgAttest(address string, cid string, q *queue.UploadQueue) (upload types.Upload, err error) {
	msg := storageTypes.NewMsgAttest(address, cid)

	if err := msg.ValidateBasic(); err != nil {
		return upload, err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	upload = types.Upload{
		Message:  msg,
		Err:      nil,
		Callback: &wg,
		Response: nil,
	}

	q.Append(&upload)
	return
}

func (f *FileServer) attest(w *http.ResponseWriter, r *http.Request) {
	var attest types.AttestRequest

	err := json.NewDecoder(r.Body).Decode(&attest)
	if err != nil {
		fmt.Println("Attest request was malformed.")
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	dealReq := &storageTypes.QueryActiveDealRequest{
		Cid: attest.Cid,
	}

	fmt.Printf("Attesting for: %s\n", attest.Cid)

	deal, err := f.queryClient.ActiveDeals(context.Background(), dealReq)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
	}

	verified, err := verifyAttest(deal.ActiveDeals, attest)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	if !verified {
		http.Error(*w, errors.New("failed to verify attest").Error(), http.StatusBadRequest)
	}

	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	upload, err := addMsgAttest(address, attest.Cid, f.queue)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	upload.Callback.Wait()

	if upload.Err != nil {
		http.Error(*w, upload.Err.Error(), http.StatusBadRequest)
		return
	}

	if upload.Response.Code != 0 {
		http.Error(*w, fmt.Errorf(upload.Response.RawLog).Error(), http.StatusBadRequest)
		return
	}

	v := types.ProxyResponse{
		Ok: true,
	}

	err = json.NewEncoder(*w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
