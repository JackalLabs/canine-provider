package jprovd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/JackalLabs/jackal-provider/jprovd/queue"
	"github.com/JackalLabs/jackal-provider/jprovd/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	storagetypes "github.com/jackal-dao/canine/x/storage/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func listqueue(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, q *queue.UploadQueue) {
	messages := make([]sdk.Msg, 0)

	for _, v := range q.Queue {
		messages = append(messages, v.Message)
	}

	v := types.QueueResponse{
		Messages: messages,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func listFiles(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	file, err := cmd.Flags().GetString("storagedir")
	if err != nil {
		return
	}

	files, _ := os.ReadDir(fmt.Sprintf("%s/networkfiles/%s/", file, ps.ByName("file")))

	var fileNames []string = make([]string, 0)

	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}

	v := types.ListResponse{
		Files: fileNames,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func dumpdb(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, db *leveldb.DB) {
	data := make([]types.DataBlock, 0)
	iter := db.NewIterator(nil, nil)

	for iter.Next() {
		d := types.DataBlock{
			Key:   string(iter.Key()),
			Value: string(iter.Value()),
		}
		data = append(data, d)
	}

	v := types.DBResponse{
		Data: data,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func showDeals(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, db *leveldb.DB) {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryClient := storagetypes.NewQueryClient(clientCtx)

	params := &storagetypes.QueryAllActiveDealsRequest{}

	res, err := queryClient.ActiveDealsAll(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}

	v := types.DealsResponse{
		Deals: res.ActiveDeals,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func showStrays(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, db *leveldb.DB) {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryClient := storagetypes.NewQueryClient(clientCtx)

	params := &storagetypes.QueryAllStraysRequest{}

	res, err := queryClient.StraysAll(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}

	v := types.StraysResponse{
		Strays: res.Strays,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func BuildApi(cmd *cobra.Command, q *queue.UploadQueue, router *httprouter.Router, db *leveldb.DB) {
	lres := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		listFiles(cmd, w, r, ps)
	}

	queue := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		listqueue(cmd, w, r, ps, q)
	}

	dumpdb := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		dumpdb(cmd, w, r, ps, db)
	}

	dealreq := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		showDeals(cmd, w, r, ps, db)
	}

	straysreq := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		showStrays(cmd, w, r, ps, db)
	}

	// CLIENT
	router.GET("/api/client/list", lres)
	router.GET("/api/client/l", lres)
	router.GET("/api/client/queue", queue)
	router.GET("/api/client/q", queue)

	// DATA
	router.GET("/api/data/dump", dumpdb)

	// NETWORK
	router.GET("/api/network/deals", dealreq)
	router.GET("/api/network/strays", straysreq)
}
