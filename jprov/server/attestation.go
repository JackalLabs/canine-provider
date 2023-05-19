package server

import (
	"context"
	"encoding/json"
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
	"github.com/syndtr/goleveldb/leveldb"
)

func attest(w *http.ResponseWriter, r *http.Request, cmd *cobra.Command, db *leveldb.DB, q *queue.UploadQueue) {
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

	queryClient := storageTypes.NewQueryClient(clientCtx)

	dealReq := &storageTypes.QueryActiveDealRequest{
		Cid: attest.Cid,
	}

	deal, err := queryClient.ActiveDeals(context.Background(), dealReq)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	merkle := deal.ActiveDeals.Merkle
	block := deal.ActiveDeals.Blocktoprove
	blockNum, err := strconv.ParseInt(block, 10, 64)
	if err != nil {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
	}

	verified := storageKeeper.VerifyDeal(merkle, attest.HashList, blockNum, attest.Item)

	if !verified {
		http.Error(*w, err.Error(), http.StatusBadRequest)
		return
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
}
