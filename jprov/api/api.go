package api

import (
	"net/http"

	"github.com/JackalLabs/jackal-provider/jprov/api/client"
	"github.com/JackalLabs/jackal-provider/jprov/api/data"
	"github.com/JackalLabs/jackal-provider/jprov/api/network"
	"github.com/JackalLabs/jackal-provider/jprov/queue"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func BuildApi(cmd *cobra.Command, q *queue.UploadQueue, router *httprouter.Router, db *leveldb.DB) {
	// CLIENT
	router.GET("/api/client/list", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		client.ListFiles(cmd, w, r, ps)
	})
	router.GET("/api/client/queue", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		client.ListQueue(cmd, w, r, ps, q)
	})
	router.GET("/api/client/space", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		client.GetSpace(cmd, w, r, ps)
	})

	// DATA
	router.GET("/api/data/dump", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		data.DumpDB(cmd, w, r, ps, db)
	})

	// NETWORK
	router.GET("/api/network/deals", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		network.ShowDeals(cmd, w, r, ps, db)
	})
	router.GET("/api/network/strays", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		network.ShowStrays(cmd, w, r, ps, db)
	})
	router.GET("/api/network/balance", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		network.GetBalance(cmd, w, r, ps)
	})
}
