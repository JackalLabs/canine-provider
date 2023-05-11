package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/julienschmidt/httprouter"

	"github.com/JackalLabs/jackal-provider/jprov/api"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/spf13/cobra"
)

func indexres(cmd *cobra.Command, w http.ResponseWriter) {
	clientCtx, err := client.GetClientTxContext(cmd)
	ctx := utils.GetServerContextFromCmd(cmd)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}

	v := types.IndexResponse{
		Status:  "online",
		Address: address,
	}
	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		ctx.Logger.Error(err.Error())
	}
}

func checkVersion(cmd *cobra.Command, w http.ResponseWriter, ctx *utils.Context) {
	res, err := cmd.Flags().GetString(types.VersionFlag)
	if err != nil {
		ctx.Logger.Error(err.Error())
	}

	clientCtx, error := client.GetClientTxContext(cmd)
	if error != nil {
		ctx.Logger.Error(err.Error())
		return
	}
	chainID := clientCtx.ChainID

	v := types.VersionResponse{
		Version: version.Version,
		ChainID: chainID,
	}
	if len(res) > 0 {
		v.Version = res
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		ctx.Logger.Error(err.Error())
	}
}

func downfil(cmd *cobra.Command, w http.ResponseWriter, ps httprouter.Params, ctx *utils.Context) {
	clientCtx := client.GetClientContextFromCmd(cmd)
	chunkSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		fmt.Println(err)
		return
	}

	var fileList []*[]byte

	var dataLength int

	var i int
	for { // loop through every file in the directory and fail once it hits a file that it can't find
		path := filepath.Join(utils.GetStoragePath(clientCtx, ps.ByName("file")), fmt.Sprintf("%d.jkl", i))
		f, err := os.ReadFile(path)
		if err != nil {
			break
		}
		fileList = append(fileList, &f)
		dataLength += len(f)
		i++
	}

	data := make([]byte, dataLength)

	for i, file := range fileList {
		for k, b := range *file {
			data[i*int(chunkSize)+k] = b
		}
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	_, err = w.Write(data)
	if err != nil {
		return
	}
}

func GetRoutes(cmd *cobra.Command, router *httprouter.Router, db *leveldb.DB, q *queue.UploadQueue) {
	ctx := utils.GetServerContextFromCmd(cmd)

	dfil := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		downfil(cmd, w, ps, ctx)
	}

	ires := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		indexres(cmd, w)
	}

	router.GET("/version", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		checkVersion(cmd, w, ctx)
	})
	router.GET("/download/:file", dfil)

	api.BuildApi(cmd, q, router, db)

	router.GET("/", ires)
}

func PostRoutes(cmd *cobra.Command, router *httprouter.Router, db *leveldb.DB, q *queue.UploadQueue) {
	upfil := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		fileUpload(&w, r, cmd, db, q)
	}

	router.POST("/upload", upfil)
	router.POST("/u", upfil)
}

func PProfRoutes(router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, "/debug/pprof/", pprof.Index)
	router.HandlerFunc(http.MethodGet, "/debug/pprof/cmdline", pprof.Cmdline)
	router.HandlerFunc(http.MethodGet, "/debug/pprof/profile", pprof.Profile)
	router.HandlerFunc(http.MethodGet, "/debug/pprof/symbol", pprof.Symbol)
	router.HandlerFunc(http.MethodGet, "/debug/pprof/trace", pprof.Trace)
	router.Handler(http.MethodGet, "/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handler(http.MethodGet, "/debug/pprof/heap", pprof.Handler("heap"))
	router.Handler(http.MethodGet, "/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handler(http.MethodGet, "/debug/pprof/block", pprof.Handler("block"))
}

// This function returns the filename(to save in database) of the saved file
// or an error if it occurs
func fileUpload(w *http.ResponseWriter, r *http.Request, cmd *cobra.Command, db *leveldb.DB, q *queue.UploadQueue) {
	ctx := utils.GetServerContextFromCmd(cmd)

	// ParseMultipartForm parses a request body as multipart/form-data
	err := r.ParseMultipartForm(types.MaxFileSize) // MAX file size lives here
	if err != nil {
		ctx.Logger.Error("Error with parsing form!")
		v := types.ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
		if err != nil {
			ctx.Logger.Error(err.Error())
		}
		return
	}
	sender := r.Form.Get("sender")

	file, handler, err := r.FormFile("file") // Retrieve the file from form data
	if err != nil {
		ctx.Logger.Error("Error with form file!")
		v := types.ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
		if err != nil {
			ctx.Logger.Error(err.Error())
		}
		return
	}

	err = saveFile(file, handler, sender, cmd, db, w, q)
	if err != nil {
		v := types.ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
		if err != nil {
			ctx.Logger.Error(err.Error())
		}
	}
}
