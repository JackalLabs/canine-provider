package queue

import (
	"fmt"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"

	"github.com/cosmos/cosmos-sdk/client"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
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
	for _, item := range q.Queue {
		if item.Message == upload.Message {
			return
		}
	}

	q.Queue = append(q.Queue, upload)
}

func (q *UploadQueue) listenOnce(cmd *cobra.Command, providerName string) {
	if q.Locked {
		return
	}
	q.Locked = true
	defer func() {
		q.Locked = false
	}()

	ctx := utils.GetServerContextFromCmd(cmd)

	l := len(q.Queue)

	if l == 0 {
		return
	}

	maxSize, err := cmd.Flags().GetInt(types.FlagMessageSize)
	if err != nil {
		ctx.Logger.Error(err.Error())
	}

	var totalSizeOfMsgs int
	msgs := make([]cosmosTypes.Msg, 0)
	uploads := make([]*types.Upload, 0)

	for i := 0; i < l; i++ { // loop through entire queue

		upload := q.Queue[i]

		uploadSize := len(upload.Message.String())

		// if the size of the upload would put us past our cap, we cut off the queue and send only what fits
		if totalSizeOfMsgs+uploadSize > maxSize {
			msgs = msgs[:len(msgs)-1]
			uploads = uploads[:len(uploads)-1]
			l = i

			break
		} else {
			uploads = append(uploads, upload)
			msgs = append(msgs, upload.Message)
			totalSizeOfMsgs += len(upload.Message.String())
		}

	}

	clientCtx := client.GetClientContextFromCmd(cmd)
	ctx.Logger.Info(fmt.Sprintf("TOTAL no. of msgs in proof transaction is: %d", len(msgs)))

	res, err := utils.SendTx(clientCtx, cmd.Flags(), fmt.Sprintf("Storage Provided by %s", providerName), msgs...)
	for _, v := range uploads {
		if v == nil {
			continue
		}
		if err != nil {
			v.Err = err
		} else {
			if res != nil {
				if res.Code != 0 {
					v.Err = fmt.Errorf(res.RawLog)
				} else {
					v.Response = res
				}
			}
		}
		if v.Callback != nil {
			v.Callback.Done()
		}
	}

	q.Queue = q.Queue[l:] // pop every upload that fit off the queue
}

func (q *UploadQueue) StartListener(cmd *cobra.Command, providerName string) {
	for {
		interval, err := cmd.Flags().GetInt64(types.FlagQueueInterval)
		if err != nil {
			interval = 2
		}
		time.Sleep(time.Second * time.Duration(interval))

		q.listenOnce(cmd, providerName)
	}
}
