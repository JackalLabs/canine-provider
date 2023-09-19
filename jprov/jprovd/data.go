package main

import (
	"encoding/json"
	"fmt"

	apitypes "github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
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

            db, err := archive.NewDoubleRefArchiveDB(utils.GetArchiveDBPath(clientCtx))
			if err != nil {
				fmt.Println(err)
				return
			}

			data := make([]apitypes.DataBlock, 0)
			iter := db.NewIterator()

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
