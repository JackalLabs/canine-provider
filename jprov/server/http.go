package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"

	"github.com/cosmos/cosmos-sdk/version"

	"github.com/julienschmidt/httprouter"

	"github.com/JackalLabs/jackal-provider/jprov/api"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/types"
)

func (f *FileServer) indexres(w http.ResponseWriter) {
	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		f.serverCtx.Logger.Error(err.Error())
		return
	}

	v := types.IndexResponse{
		Status:  "online",
		Address: address,
	}
	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		f.serverCtx.Logger.Error(err.Error())
	}
}

func (f *FileServer) checkVersion(w http.ResponseWriter) {
	res, err := f.cmd.Flags().GetString(types.VersionFlag)
	if err != nil {
		f.serverCtx.Logger.Error(err.Error())
	}

	v := types.VersionResponse{
		Version: version.Version,
		ChainID: f.cosmosCtx.ChainID,
	}
	if len(res) > 0 {
		v.Version = res
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		f.serverCtx.Logger.Error(err.Error())
	}
}

func (f *FileServer) downfil(w http.ResponseWriter, ps httprouter.Params) {
	fid := string(ps.ByName("file"))
	file, err := f.archive.RetrieveFile(fid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			f.serverCtx.Logger.Error("downfil: %s", err)
		}
	}()

	written, err := io.Copy(w, file)
	if err != nil {
		f.serverCtx.Logger.Error("downfil: %s", err)
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", written))
}

func (f *FileServer) GetRoutes(router *httprouter.Router) {
	dfil := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		f.downfil(w, ps)
	}

	ires := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		f.indexres(w)
	}

	router.GET("/version", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		f.checkVersion(w)
	})
	router.GET("/download/:file", dfil)

	api.BuildApi(f.cmd, f.queue, router, f.archivedb, f.downtimedb)

	router.GET("/", ires)
}

func (f *FileServer) PostRoutes(router *httprouter.Router) {
	upfil := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		f.fileUpload(&w, r)
	}

	router.POST("/upload", upfil)
	router.POST("/u", upfil)

	router.POST("/attest", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		f.attest(&w, r)
	})
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
func (f  *FileServer) fileUpload(w *http.ResponseWriter, r *http.Request) {
	// ParseMultipartForm parses a request body as multipart/form-data
	err := r.ParseMultipartForm(types.MaxFileSize) // MAX file size lives here
	if err != nil {
		f.serverCtx.Logger.Error("Error with parsing form!")
		v := types.ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
		if err != nil {
			f.serverCtx.Logger.Error(err.Error())
		}
		return
	}
	sender := r.Form.Get("sender")

	file, handler, err := r.FormFile("file") // Retrieve the file from form data
	if err != nil {
		f.serverCtx.Logger.Error("Error with form file!")
		v := types.ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
		if err != nil {
			f.serverCtx.Logger.Error(err.Error())
		}
		return
	}

	err = f.handleUploadRequest(file, handler, sender, w)
	if err != nil {
		v := types.ErrorResponse{
			Error: err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
		if err != nil {
			f.serverCtx.Logger.Error(err.Error())
		}
	}
}
