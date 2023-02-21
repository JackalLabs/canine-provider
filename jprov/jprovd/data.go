package main

import (
	"encoding/json"
	"fmt"

	apitypes "github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func CmdDumpDatabase() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump database contents to console.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			path := utils.GetDataPath(clientCtx)

			db, dberr := leveldb.OpenFile(path, nil)
			if dberr != nil {
				fmt.Println(dberr)
				return
			}

			data := make([]apitypes.DataBlock, 0)
			iter := db.NewIterator(nil, nil)

			for iter.Next() {
				d := apitypes.DataBlock{
					Key:   string(iter.Key()),
					Value: string(iter.Value()),
				}
				data = append(data, d)
			}

			v := apitypes.DBResponse{
				Data: data,
			}

			r, err := json.Marshal(v)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println(string(r))

			return err
		},
	}

	return cmd
}
