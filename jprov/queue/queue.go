package queue

import (
	"fmt"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/testutils"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"

	"github.com/cosmos/cosmos-sdk/client"
	ctypes "github.com/cosmos/cosmos-sdk/types"
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

func (q *UploadQueue) listenOnce(cmd *cobra.Command) {
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

	logger, logFile := testutils.CreateLogger("cutTheQueue")
	logger.Println("===============BROADCASTING NEW BATCH====================")

	logger.Printf("length of queue is : %d\n", l)

	msgs := make([]ctypes.Msg, 0)
	uploads := make([]*types.Upload, 0)

	for i := 0; i < l; i++ {
		totalSizeOfMsgs := 0 // keep track of total messages size estimate

		for _, m := range msgs {
			logger.Printf("len(m.String() is?: %d --bigger than 1?\n", len(m.String()))
			totalSizeOfMsgs += len(m.String())
		}

		upload := q.Queue[i]

		uploadSize := len(upload.Message.String())
		logger.Printf("totalSizeOfMsgs is now : %d --getting bigger?\n", l)

		// if the size of the upload would put us past our cap, we cut off the queue and send only what fits
		if totalSizeOfMsgs+uploadSize > maxSize {
			logger.Printf("totalSizeOfMsgs+uploadSize is : %d, which is bigger than %d\n", l, maxSize)
			msgs = msgs[:len(msgs)-1]
			l = i
			break
		}

		uploads = append(uploads, upload)
		msgs = append(msgs, upload.Message)
		// ctx.Logger.Info(fmt.Sprintf("Message being sent to chain: %s", upload.Message.String()))

	}
	logFile.Close()

	clientCtx := client.GetClientContextFromCmd(cmd)

	res, err := utils.SendTx(clientCtx, cmd.Flags(), msgs...)
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

func (q *UploadQueue) StartListener(cmd *cobra.Command) {
	for {
		time.Sleep(time.Second * 2)

		q.listenOnce(cmd)
	}
}
