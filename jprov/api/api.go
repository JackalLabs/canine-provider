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
	lres := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		client.ListFiles(cmd, w, r, ps)
	}

	queue := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		client.ListQueue(cmd, w, r, ps, q)
	}

	dumpdb := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		data.DumpDB(cmd, w, r, ps, db)
	}

	dealreq := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		network.ShowDeals(cmd, w, r, ps, db)
	}

	straysreq := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		network.ShowStrays(cmd, w, r, ps, db)
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
