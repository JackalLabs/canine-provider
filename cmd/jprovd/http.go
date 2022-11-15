package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/julienschmidt/httprouter"

	storagetypes "github.com/jackal-dao/canine/x/storage/types"
	"github.com/spf13/cobra"
)

func indexres(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	address := clientCtx.GetFromAddress()

	v := IndexResponse{
		Status:  "online",
		Address: address.String(),
	}
	json.NewEncoder(w).Encode(v)
}

func checkVersion(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	v := VersionResponse{
		Version: "1.0.0",
	}
	json.NewEncoder(w).Encode(v)
}

func downfil(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	file, err := cmd.Flags().GetString("storagedir")
	if err != nil {
		return
	}

	files, _ := os.ReadDir(fmt.Sprintf("%s/networkfiles/%s/", file, ps.ByName("file")))

	var data []byte

	for i := 0; i < len(files); i += 1 {
		f, err := os.ReadFile(fmt.Sprintf("%s/networkfiles/%s/%d%s", file, ps.ByName("file"), i, ".jkl"))
		if err != nil {
			fmt.Printf("Error can't open file!\n")
			w.Write([]byte("cannot find file"))
			return
		}

		data = append(data, f...)
	}

	w.Write(data)
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

	v := ListResponse{
		Files: fileNames,
	}

	json.NewEncoder(w).Encode(v)
}

func dumpdb(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, db *leveldb.DB) {
	data := make([]DataBlock, 0)
	iter := db.NewIterator(nil, nil)

	for iter.Next() {
		d := DataBlock{
			Key:   string(iter.Key()),
			Value: string(iter.Value()),
		}
		data = append(data, d)
	}

	v := DBResponse{
		Data: data,
	}

	json.NewEncoder(w).Encode(v)
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

	v := DealsResponse{
		Deals: res.ActiveDeals,
	}

	json.NewEncoder(w).Encode(v)
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

	v := StraysResponse{
		Strays: res.Strays,
	}

	json.NewEncoder(w).Encode(v)
}

func (q *UploadQueue) buildApi(cmd *cobra.Command, router *httprouter.Router, db *leveldb.DB) {
	lres := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		listFiles(cmd, w, r, ps)
	}

	queue := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		q.listqueue(cmd, w, r, ps)
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

func (q *UploadQueue) getRoutes(cmd *cobra.Command, router *httprouter.Router, db *leveldb.DB) {
	dfil := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		downfil(cmd, w, r, ps)
	}

	ires := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		indexres(cmd, w, r, ps)
	}

	router.GET("/version", checkVersion)
	router.GET("/v", checkVersion)
	router.GET("/download/:file", dfil)
	router.GET("/d/:file", dfil)

	q.buildApi(cmd, router, db)

	router.GET("/", ires)
}

func (q *UploadQueue) postRoutes(cmd *cobra.Command, router *httprouter.Router, db *leveldb.DB) {
	upfil := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		q.fileUpload(&w, r, ps, cmd, db)
	}

	router.POST("/upload", upfil)
	router.POST("/u", upfil)
}

func (q *UploadQueue) listqueue(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	messages := make([]types.Msg, 0)

	for _, v := range q.Queue {
		messages = append(messages, v.Message)
	}

	v := QueueResponse{
		Messages: messages,
	}

	json.NewEncoder(w).Encode(v)
}

// This function returns the filename(to save in database) of the saved file
// or an error if it occurs
func (q *UploadQueue) fileUpload(w *http.ResponseWriter, r *http.Request, ps httprouter.Params, cmd *cobra.Command, db *leveldb.DB) {
	// ParseMultipartForm parses a request body as multipart/form-data
	r.ParseMultipartForm(MaxFileSize) // MAX file size lives here

	file, handler, err := r.FormFile("file") // Retrieve the file from form data

	sender := r.Form.Get("sender")

	if err != nil {
		fmt.Printf("Error with form file!\n")
		v := ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(*w).Encode(v)
		return
	}

	err = q.saveFile(file, handler, sender, cmd, db, w)
	if err != nil {
		v := ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(*w).Encode(v)
	}
}
