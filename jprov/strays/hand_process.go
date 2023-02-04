package strays

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func (h *LittleHand) ProcessStray() {
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
}

func (m *StrayManager) CollectStrays(cmd *cobra.Command) {
	qClient := storageTypes.NewQueryClient(m.ClientContext)

	res, err := qClient.StraysAll(cmd.Context(), &storageTypes.QueryAllStraysRequest{})
	if err != nil {
		m.Context.Logger.Error(err.Error())
		return
	}
	s := res.Strays

	if len(s) == 0 { // If there are no strays, the network has claimed them all. We will try again later.
		return
	}

	req := storageTypes.QueryProviderRequest{ // Ask the network what my own IP address is registered to.
		Address: m.Address,
	}

	provs, err := qClient.Providers(cmd.Context(), &req) // Publish the ask.
	if err != nil {
		m.Context.Logger.Error(err.Error())
		return
	}
	ip := provs.Providers.Address // Our IP address

	for _, newStray := range s { // Only add new strays to the queue
		clean := true
		for _, oldStray := range m.Strays {
			if newStray.Cid == oldStray.Cid {
				clean = false
			}
		}
		for _, hands := range m.hands { // check active processes too
			if newStray.Cid == hands.Stray.Cid {
				clean = false
			}
		}
		if clean {
			m.Strays = append(m.Strays, &newStray)
		}
	}
}

func (q *UploadQueue) CheckStrays(cmd *cobra.Command, db *leveldb.DB) {
	for {
		time.Sleep(time.Second * 5)

		q.checkStraysOnce(cmd, db)

	}
}
