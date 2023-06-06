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
	"github.com/cosmos/cosmos-sdk/client"
	storageKeeper "github.com/jackalLabs/canine-chain/x/storage/keeper"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/spf13/cobra"
)

func verifyAttest (ctx client.Context, attest types.AttestRequest) (verified bool, err error) {
	queryClient := storageTypes.NewQueryClient(ctx)

	dealReq := &storageTypes.QueryActiveDealRequest{
		Cid: attest.Cid,
	}

	deal, err := queryClient.ActiveDeals(context.Background(), dealReq)
	if err != nil {
		return false, err
	}

	merkle := deal.ActiveDeals.Merkle
    block := deal.ActiveDeals.Blocktoprove
    blockNum, err := strconv.ParseInt(block, 10, 64)
	if err != nil {
		return false, err
	}

	verified = storageKeeper.VerifyDeal(merkle, attest.HashList, blockNum, attest.Item)

	return
}

func sendAttestMsg(ctx client.Context, cid string, q *queue.UploadQueue) (upload types.Upload, err error) {
	address, err := crypto.GetAddress(ctx)
	if err != nil {
		return upload, err
	}

	msg := storageTypes.NewMsgAttest(address, cid)

	if err := msg.ValidateBasic(); err != nil {
		return upload, err
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

	return
}

func attest(w *http.ResponseWriter, r *http.Request, cmd *cobra.Command, q *queue.UploadQueue) {
	clientCtx, qerr := client.GetClientTxContext(cmd)
	if qerr != nil {
		fmt.Println(qerr)
		return
	}

	var attest types.AttestRequest

	err := json.NewDecoder(r.Body).Decode(&attest)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	verified, err := verifyAttest(clientCtx, attest)

	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	if !verified {
		http.Error(*w, errors.New("failed to verify attest").Error(), http.StatusBadRequest)
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := storageTypes.NewMsgAttest( // create new attest
		address,
		attest.Cid,
	)
	if err := msg.ValidateBasic(); err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
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
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	if u.Response.Code != 0 {
		http.Error(*w, fmt.Errorf(u.Response.RawLog).Error(), http.StatusBadRequest)
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
