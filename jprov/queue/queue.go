package queue

import (
	"encoding/json"
	"fmt"
	"os"
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

func (q *UploadQueue) checkStraysOnce(cmd *cobra.Command, db *leveldb.DB) {
	clientCtx := client.GetClientContextFromCmd(cmd)

	ctx := utils.GetServerContextFromCmd(cmd)

	qClient := storageTypes.NewQueryClient(clientCtx)

	res, err := qClient.StraysAll(cmd.Context(), &storageTypes.QueryAllStraysRequest{})
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}

	s := res.Strays

	if len(s) == 0 {
		return
	}

	for _, stray := range s {

		if _, err := os.Stat(utils.GetStoragePath(clientCtx, stray.Fid)); !os.IsNotExist(err) {
			continue
		}

		filesres, err := qClient.FindFile(cmd.Context(), &storageTypes.QueryFindFileRequest{Fid: stray.Fid})
		if err != nil {
			ctx.Logger.Error(err.Error())
			continue
			// return err
		}
		ctx.Logger.Info(filesres.ProviderIps)

		var arr []string
		err = json.Unmarshal([]byte(filesres.ProviderIps), &arr)
		if err != nil {
			ctx.Logger.Error(err.Error())
			continue
		}

		if len(arr) == 0 {
			err = fmt.Errorf("no providers have the file we want something is wrong")
			ctx.Logger.Error(err.Error())
			continue
		}

		_, err = utils.DownloadFileFromURL(cmd, arr[0], stray.Fid, stray.Cid, db, ctx.Logger)
		if err != nil {
			ctx.Logger.Error(err.Error())
			continue
		}

		address, err := crypto.GetAddress(clientCtx)
		if err != nil {
			ctx.Logger.Error(err.Error())
			continue
		}

		msg := storageTypes.NewMsgClaimStray(
			address,
			stray.Cid,
		)
		if err := msg.ValidateBasic(); err != nil {
			ctx.Logger.Error(err.Error())
			continue
		}

		u := types.Upload{
			Message:  msg,
			Callback: nil,
			Err:      nil,
			Response: nil,
		}

		q.Queue = append(q.Queue, &u)

		ctx.Logger.Info(res.String())
	}
}

func (q *UploadQueue) CheckStrays(cmd *cobra.Command, db *leveldb.DB) {
	for {
		time.Sleep(time.Second)

		q.checkStraysOnce(cmd, db)

	}
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
