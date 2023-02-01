package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func GetStatus(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientCtx, err := sdkclient.GetClientTxContext(cmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	node, err := clientCtx.GetNode()
	if err != nil {
		return
	}

	s, err := node.Status(context.Background())
	if err != nil {
		return
	}

	err = json.NewEncoder(w).Encode(s)
	if err != nil {
		fmt.Println(err)
	}
}
