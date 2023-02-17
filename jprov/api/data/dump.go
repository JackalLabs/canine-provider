package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/syndtr/goleveldb/leveldb"
)

func DumpDB(w http.ResponseWriter, db *leveldb.DB) {
	data := make([]types.DataBlock, 0)
	iter := db.NewIterator(nil, nil)

	for iter.Next() {
		if string(iter.Key())[:4] == "TREE" {
			continue
		}
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

func DumpDowntimes(w http.ResponseWriter, db *leveldb.DB) {
	data := make([]types.DowntimeBlock, 0)
	iter := db.NewIterator(nil, nil)

	for iter.Next() {
		if string(iter.Key())[:5] == utils.DowntimeKey {
			i, err := strconv.Atoi(string(iter.Value()))
			if err != nil {
				continue
			}
			d := types.DowntimeBlock{
				CID:      string(iter.Key())[5:],
				Downtime: i,
			}
			data = append(data, d)

		}
	}

	v := types.DowntimeResponse{
		Data: data,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func DumpFids(w http.ResponseWriter, db *leveldb.DB) {
	data := make([]types.FidBlock, 0)
	iter := db.NewIterator(nil, nil)

	for iter.Next() {
		if string(iter.Key())[:5] == utils.FileKey {

			d := types.FidBlock{
				CID: string(iter.Key())[5:],
				FID: string(iter.Value()),
			}
			data = append(data, d)

		}
	}

	v := types.FidResponse{
		Data: data,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
