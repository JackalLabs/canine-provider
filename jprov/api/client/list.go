package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	provtypes "github.com/JackalLabs/jackal-provider/jprov/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func ListQueue(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params, q *queue.UploadQueue) {
	messages := make([]sdk.Msg, 0)

	for _, v := range q.Queue {
		messages = append(messages, v.Message)
	}

	v := types.QueueResponse{
		Messages: messages,
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func ListFiles(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	file, err := cmd.Flags().GetString(provtypes.DataDir)
	if err != nil {
		return
	}

	files, _ := os.ReadDir(fmt.Sprintf("%s/networkfiles/%s/", file, ps.ByName("file")))

	var fileNames []string = make([]string, 0)

	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}

	v := types.ListResponse{
		Files: fileNames,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
