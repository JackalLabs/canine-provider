package data

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func DumpDB(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, db *leveldb.DB) {
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
