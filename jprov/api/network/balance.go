package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func GetBalance(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientCtx, err := sdkclient.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryClient := banktypes.NewQueryClient(clientCtx)
	address := clientCtx.GetFromAddress()

	params := &banktypes.QueryBalanceRequest{
		Denom:   "ujkl",
		Address: address.String(),
	}

	res, err := queryClient.Balance(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}

	v := types.BalanceResponse{
		Balance: res.Balance,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
