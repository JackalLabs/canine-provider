package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"

	ctypes "github.com/cosmos/cosmos-sdk/types"
)

type UploadQueue struct {
	Queue  []*types.Upload
	Locked bool
}

func New() UploadQueue {
	queue := UploadQueue{
		Queue:  make([]*types.Upload, 0),
		Locked: false,
	}
	return queue
}

func (q *UploadQueue) Append(upload *types.Upload) {
	q.Queue = append(q.Queue, upload)
}

func (q *UploadQueue) listenOnce(cmd *cobra.Command) {
	if q.Locked {
		return
	}

	l := len(q.Queue)

	if l == 0 {
		return
	}

	msg := make([]ctypes.Msg, 0)
	uploads := make([]*types.Upload, 0)
	for i := 0; i < l; i++ {
		upload := q.Queue[i]
		uploads = append(uploads, upload)
		msg = append(msg, upload.Message)
	}

	clientCtx := client.GetClientContextFromCmd(cmd)

	res, err := utils.SendTx(clientCtx, cmd.Flags(), msg...)
	for _, v := range uploads {
		if err != nil {
			v.Err = err
		} else {
			if res.Code != 0 {
				v.Err = fmt.Errorf(res.RawLog)
			} else {
				v.Response = res
			}
		}
		if v.Callback != nil {
			v.Callback.Done()
		}
	}

	q.Queue = q.Queue[l:]
}

func (q *UploadQueue) StartListener(cmd *cobra.Command) {
	for {
		time.Sleep(time.Second * 2)

		q.listenOnce(cmd)
	}
}
