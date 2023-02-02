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

	if len(s) == 0 { // If there are no strays, the network has claimed them all. We will try again later.
		return
	}

	addr, err := crypto.GetAddress(clientCtx) // Getting the address of the provider to compare it to the strays.
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}
	req := storageTypes.QueryProviderRequest{ // Ask the network what my own IP address is registered to.
		Address: addr,
	}

	provs, err := qClient.Providers(cmd.Context(), &req) // Publish the ask.
	if err != nil {
		ctx.Logger.Error(err.Error())
		return
	}
	ip := provs.Providers.Address // Our IP address

	for _, stray := range s { // For every stray, we try and claim & download it.
		ctx.Logger.Info(fmt.Sprintf("Getting info for %s", stray.Cid))

		filesres, err := qClient.FindFile(cmd.Context(), &storageTypes.QueryFindFileRequest{Fid: stray.Fid}) // List all providers that currently have the file active.
		if err != nil {
			ctx.Logger.Error(err.Error())
			continue // There was an issue, so we pretend like it didn't happen.
		}

		var arr []string // Create an array of IPs from the request.
		err = json.Unmarshal([]byte(filesres.ProviderIps), &arr)
		if err != nil {
			ctx.Logger.Error(err.Error())
			continue
		}

		if len(arr) == 0 {
			/**
			If there are no providers with the file, we check if it's on our provider's filesystem. (We cannot claim
			strays that we don't own, but if we caused an error when handling the file we can reclaim the stray with
			the cached file from our filesystem which keeps the file alive)
			*/
			if _, err := os.Stat(utils.GetStoragePath(clientCtx, stray.Fid)); os.IsNotExist(err) {
				ctx.Logger.Info("Nobody, not even I have the file.")
				continue // If we don't have it and nobody else does, there is nothing we can do.
			}

			err = utils.SaveToDatabase(stray.Fid, stray.Cid, db, ctx.Logger) // Add the file back to the database since it's never being downloaded
			if err != nil {
				ctx.Logger.Error(err.Error())
				continue
			}
		} else { // If there are providers with this file, we will download it from them instead to keep things consistent
			if _, err := os.Stat(utils.GetStoragePath(clientCtx, stray.Fid)); !os.IsNotExist(err) {
				ctx.Logger.Info("Already have this file")
				continue
			}

			found := false
			for _, prov := range arr { // Check every provider for the file, not just trust chain data.
				if prov == ip { // Ignore ourselves
					continue
				}
				_, err = utils.DownloadFileFromURL(cmd, prov, stray.Fid, stray.Cid, db, ctx.Logger)
				if err != nil {
					ctx.Logger.Error(err.Error())
					continue
				}
				found = true // If we can successfully download the file, stop there.
				break
			}

			if !found { // If we never find the file, and we don't have it, something is wrong with the network, nothing we can do.
				ctx.Logger.Info("Cannot find the file we want, either something is wrong or you have the file already")
				continue
			}
		}

		ctx.Logger.Info(fmt.Sprintf("Attempting to claim %s on chain", stray.Cid))

		msg := storageTypes.NewMsgClaimStray( // Attempt to claim the stray, this may fail if someone else has already tried to claim our stray.
			addr,
			stray.Cid,
		)
		if err := msg.ValidateBasic(); err != nil {
			ctx.Logger.Error(err.Error())
			continue
		}

		var wg sync.WaitGroup
		wg.Add(1)

		u := types.Upload{
			Message:  msg,
			Callback: &wg,
			Err:      nil,
			Response: nil,
		}

		q.Queue = append(q.Queue, &u)

		wg.Wait()

		e := ""
		if u.Err != nil {
			e = u.Err.Error()
		}

		r := ""
		if u.Response != nil {
			r = u.Response.String()
		}
		_ = r
		_ = e

		// ctx.Logger.Info(fmt.Sprintf("CID: %s, FID: %s, Error: %s, Res: %s", stray.Cid, stray.Fid, e, r))
		break
	}
}

func (q *UploadQueue) CheckStrays(cmd *cobra.Command, db *leveldb.DB) {
	for {
		time.Sleep(time.Second * 5)

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
	res, err := cmd.Flags().GetBool(types.HaltStraysFlag)
	if err != nil {
		fmt.Println(err)
		return
	}
	if res {
		return
	}
	for {
		time.Sleep(time.Second * 2)

		q.listenOnce(cmd)
	}
}
