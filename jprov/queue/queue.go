package queue

import (
	"errors"
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

// Return a list of messages of the upload queue up to maxMessageSize in FCFS order.
// Returns nil in these conditions:
//  1. maxMessageSize is too small
//  2. the UploadQueue is locked
//  3. the upload queue is empty
func (q *UploadQueue) PrepareMessage(maxMessageSize int) (messages []cosmosTypes.Msg) {
	if maxMessageSize < 1 || !q.Locked || len(q.Queue) == 0 {
		return nil
	}

	var netMsgSize int
	for _, q := range q.Queue {
		msgSize := len(q.Message.String())

		if netMsgSize+msgSize > maxMessageSize {
			break
		} else {
			netMsgSize += msgSize
			messages = append(messages, q.Message)
		}
	}

	return
}

// Update the upload queue with the parameter fields of count
func (q *UploadQueue) UpdateQueue(count int, err error, res *cosmosTypes.TxResponse) {
	if !q.Locked || len(q.Queue) == 0 || len(q.Queue) < count {
		return
	}

	for i := 0; i < count; i++ {
		q := q.Queue[i]

		if err != nil {
			q.Err = err
		} else {
			if res != nil {
				if res.Code != 0 {
					q.Err = errors.New(res.RawLog)
				} else {
					q.Response = res
				}
			}
		}

		if q.Callback != nil {
			q.Callback.Done()
		}
	}
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

	if len(q.Queue) == 0 {
		return
	}

	maxSize, err := cmd.Flags().GetInt(types.FlagMessageSize)
	if err != nil {
		ctx.Logger.Error(err.Error())
	}

	msgs := q.PrepareMessage(maxSize)

	clientCtx := client.GetClientContextFromCmd(cmd)
	ctx.Logger.Debug(fmt.Sprintf("total no. of msgs in proof transaction is: %d", len(msgs)))

	res, err := utils.SendTx(clientCtx, cmd.Flags(), fmt.Sprintf("Storage Provided by %s", providerName), msgs...)

	q.UpdateQueue(len(msgs), err, res)

	q.Queue = q.Queue[len(msgs):] // pop every upload that fit off the queue
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
