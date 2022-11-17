package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	sdkclient "github.com/cosmos/cosmos-sdk/client"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func ShowStrays(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, db *leveldb.DB) {
	clientCtx, err := sdkclient.GetClientTxContext(cmd)
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

	v := types.StraysResponse{
		Strays: res.Strays,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
