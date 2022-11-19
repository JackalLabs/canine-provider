package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func GetSpace(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientCtx, err := sdkclient.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryClient := storagetypes.NewQueryClient(clientCtx)
	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		fmt.Println(err)
		return
	}
	params := &storagetypes.QueryGetProvidersRequest{
		Address: address,
	}
	res, err := queryClient.Providers(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		return
	}

	totalSpace := res.Providers.Totalspace

	fsparams := &storagetypes.QueryFreespaceRequest{
		Address: address,
	}
	fsres, err := queryClient.Freespace(context.Background(), fsparams)
	if err != nil {
		fmt.Println(err)
		return
	}

	freeSpace := fsres.Space

	ttint, ok := sdk.NewIntFromString(totalSpace)
	if !ok {
		return
	}
	fsint, ok := sdk.NewIntFromString(freeSpace)
	if !ok {
		return
	}

	v := types.SpaceResponse{
		Total: ttint.Int64(),
		Free:  fsint.Int64(),
		Used:  ttint.Int64() - fsint.Int64(),
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
