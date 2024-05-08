package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"

	storageKeeper "github.com/jackalLabs/canine-chain/v3/x/storage/keeper"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

func verifyAttest(deal storageTypes.LegacyActiveDeals, attest types.AttestRequest) (verified bool, err error) {
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

func (f *FileServer) handleAttestRequest(w *http.ResponseWriter, r *http.Request) error {
	var attestReq types.AttestRequest

	err := json.NewDecoder(r.Body).Decode(&attestReq)
	if err != nil {
		fmt.Println("Attest request was malformed.")
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return nil
	}

	if err != nil {
		switch err = f.attest(attestReq); {
		case errors.Is(err, errors.New("failed to verify attest")):
			http.Error(*w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, status.Error(codes.NotFound, "not found")):
			http.Error(*w, errors.New("active deal not found").Error(), http.StatusBadRequest)
		case errors.Is(err, errors.New("tx error response")):
			http.Error(*w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(*w, "internal server error", http.StatusInternalServerError)
			return err
		}
	}

	v := types.ProxyResponse{
		Ok: true,
	}

	return json.NewEncoder(*w).Encode(v)
}

func (f *FileServer) attest(attestReq types.AttestRequest) error {
	fmt.Printf("Attesting for: %s\n", attestReq.Cid)
	dealReq := &storageTypes.QueryActiveDealRequest{
		Cid: attestReq.Cid,
	}

	deal, err := f.queryClient.ActiveDeals(context.Background(), dealReq)
	if err != nil {
		return err
	}

	verified, err := verifyAttest(deal.ActiveDeals, attestReq)
	if err != nil {
		return err
	}

	if !verified {
		return errors.New("failed to verify attest")
	}

	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		return err
	}

	upload, err := addMsgAttest(address, attestReq.Cid, f.queue)
	if err != nil {
		return err
	}

	upload.Callback.Wait()

	if upload.Err != nil {
		return errors.Join(errors.New("tx error response"), err)
	}

	if upload.Response == nil {
		return errors.New("upload: no response")
	}

	if upload.Response.Code != 0 {
		return fmt.Errorf(upload.Response.RawLog)
	}

	return nil
}
