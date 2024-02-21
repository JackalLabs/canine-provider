package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	sdkclient "github.com/cosmos/cosmos-sdk/client"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func ShowDeals(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientCtx, err := sdkclient.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryClient := storagetypes.NewQueryClient(clientCtx)

	params := &storagetypes.QueryAllActiveDealsRequest{}

	res, err := queryClient.ActiveDealsAll(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}

	v := types.DealsResponse{
		Deals: res.ActiveDeals,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
