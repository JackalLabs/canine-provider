package data

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/JackalLabs/jackal-provider/jprov/archive"
)

func DumpDB(w http.ResponseWriter, db archive.ArchiveDB) {
	data := make([]types.DataBlock, 0)
	iter := db.NewIterator()

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

func DumpDowntimes(w http.ResponseWriter, db *archive.DowntimeDB) {
	data := make([]types.DowntimeBlock, 0)
	iter := db.NewIterator()

	for iter.Next() {
		downtime, err := archive.ByteToBlock(iter.Value())
		if err != nil {
			fmt.Printf("Error: DumpDowntimes(): %s", err.Error())
			continue
		}
		d := types.DowntimeBlock{
			CID:      string(iter.Key()),
			Downtime: int(downtime),
		}
		data = append(data, d)
	}

	v := types.DowntimeResponse{
		Data: data,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func DumpFids(w http.ResponseWriter, db archive.ArchiveDB) {
	data := make([]types.FidBlock, 0)
	iter := db.NewIterator()

	for iter.Next() {
		d := types.FidBlock{
			CID: string(iter.Key()),
			FID: string(iter.Value()),
		}
		data = append(data, d)
	}

	v := types.FidResponse{
		Data: data,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
