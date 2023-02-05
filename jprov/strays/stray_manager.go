package strays

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func (m *StrayManager) AddHand(db *leveldb.DB, cmd *cobra.Command, index uint) *LittleHand {
	clientCtx := client.GetClientContextFromCmd(cmd)
	pkeyStruct, err := crypto.ReadKey(clientCtx)
	if err != nil {
		return nil
	}

	key, err := indexPrivKey(pkeyStruct.Key, byte(index))
	if err != nil {
		return nil
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		return nil
	}

	hand := LittleHand{
		Waiter:        &m.Waiter,
		Stray:         nil,
		Database:      db,
		Busy:          false,
		Cmd:           cmd,
		ClientContext: clientCtx,
		Id:            index,
		Address:       address,
	}

	m.hands = append(m.hands, hand)
	return &hand
}

func (h *LittleHand) Process(ctx *utils.Context, m *StrayManager) { // process the stray and make the txn, when done, free the hand & delete the stray entry
	if h.Stray == nil {
		return
	}
	if h.Busy {
		return
	}
	h.Busy = true
	finish := func() { // macro to free up hand
		h.Stray = nil
		h.Busy = false
	}

	ctx.Logger.Info(fmt.Sprintf("Getting info for %s", h.Stray.Cid))
	qClient := storageTypes.NewQueryClient(h.ClientContext)
	filesres, err := qClient.FindFile(h.Cmd.Context(), &storageTypes.QueryFindFileRequest{Fid: h.Stray.Fid}) // List all providers that currently have the file active.
	if err != nil {
		ctx.Logger.Error(err.Error())
		finish()
		return // There was an issue, so we pretend like it didn't happen.
	}

	var arr []string // Create an array of IPs from the request.
	err = json.Unmarshal([]byte(filesres.ProviderIps), &arr)
	if err != nil {
		ctx.Logger.Error(err.Error())
		finish()
		return // There was an issue, so we pretend like it didn't happen.
	}

	if len(arr) == 0 {
		/**
		If there are no providers with the file, we check if it's on our provider's filesystem. (We cannot claim
		strays that we don't own, but if we caused an error when handling the file we can reclaim the stray with
		the cached file from our filesystem which keeps the file alive)
		*/
		if _, err := os.Stat(utils.GetStoragePath(h.ClientContext, h.Stray.Fid)); os.IsNotExist(err) {
			ctx.Logger.Info("Nobody, not even I have the file.")
			finish()
			return // If we don't have it and nobody else does, there is nothing we can do.
		}

		err = utils.SaveToDatabase(h.Stray.Fid, h.Stray.Cid, h.Database, ctx.Logger) // Add the file back to the database since it's never being downloaded
		if err != nil {
			ctx.Logger.Error(err.Error())
			finish()
			return
		}
	} else { // If there are providers with this file, we will download it from them instead to keep things consistent
		if _, err := os.Stat(utils.GetStoragePath(h.ClientContext, h.Stray.Fid)); !os.IsNotExist(err) {
			ctx.Logger.Info("Already have this file")
			finish()
			return
		}

		found := false
		for _, prov := range arr { // Check every provider for the file, not just trust chain data.
			if found {
				continue
			}
			if prov == m.Ip { // Ignore ourselves
				finish()
				return
			}
			_, err = utils.DownloadFileFromURL(h.Cmd, prov, h.Stray.Fid, h.Stray.Cid, h.Database, ctx.Logger)
			if err != nil {
				ctx.Logger.Error(err.Error())
				finish()
				return
			}
			found = true // If we can successfully download the file, stop there.
		}

		if !found { // If we never find the file, and we don't have it, something is wrong with the network, nothing we can do.
			ctx.Logger.Info("Cannot find the file we want, either something is wrong or you have the file already")
			finish()
			return
		}
	}

	ctx.Logger.Info(fmt.Sprintf("Attempting to claim %s on chain", h.Stray.Cid))

	msg := storageTypes.NewMsgClaimStray( // Attempt to claim the stray, this may fail if someone else has already tried to claim our stray.
		m.Address,
		h.Stray.Cid,
		h.Address,
	)
	if err := msg.ValidateBasic(); err != nil {
		ctx.Logger.Error(err.Error())
		finish()
		return
	}

	res, err := utils.SendTx(h.ClientContext, h.Cmd.Flags(), msg)
	if err != nil {
		ctx.Logger.Error(err.Error())
		finish()
		return
	}

	if res.Code != 0 {
		ctx.Logger.Error(res.RawLog)
	}

	finish()
}

func (m *StrayManager) Distribute() { // Hand out every available stray to an idle hand
	for i := 0; i < len(m.hands); i++ {
		h := m.hands[i]
		if h.Stray != nil { // skip all currently busy hands
			continue
		}

		if len(m.Strays) <= 0 { // make sure there are strays to distribute
			return
		}

		h.Stray, m.Strays = m.Strays[0], m.Strays[1:] // pop the first element off the queue & assign it to the hand

	}
}

func (m *StrayManager) Init(cmd *cobra.Command, count uint, db *leveldb.DB) { // create all the hands for the manager
	fmt.Println("Starting initialization...")
	var i uint
	clientCtx := client.GetClientContextFromCmd(cmd)

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		fmt.Println(err)
		return
	}

	qClient := storageTypes.NewQueryClient(clientCtx)
	pres, err := qClient.Providers(cmd.Context(), &storageTypes.QueryProviderRequest{Address: address})
	if err != nil {
		fmt.Println(err)
		return
	}

	currentClaimers := pres.Providers.AuthClaimers

	for i = 1; i < count+1; i++ {
		fmt.Printf("Processing stray thread %d.\n", i)
		h := m.AddHand(db, m.Cmd, i)

		found := false
		for _, claimer := range currentClaimers {
			if claimer == h.Address {
				found = true
				break
			}
		}
		if found {
			continue
		}

		fmt.Println("Adding hand to my claim whitelist...")

		msg := storageTypes.NewMsgAddClaimer(address, h.Address)

		res, err := utils.SendTx(clientCtx, cmd.Flags(), msg)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if res.Code != 0 {
			fmt.Println(res.RawLog)
			continue
		}
		fmt.Println("Done!")

		fmt.Println("Authorizing hand to transact on my behalf...")

		adr, nerr := sdk.AccAddressFromBech32(address)
		if nerr != nil {
			fmt.Println(nerr)
			continue
		}

		hadr, nerr := sdk.AccAddressFromBech32(h.Address)
		if nerr != nil {
			fmt.Println(nerr)
			continue
		}

		allowance := feegrant.BasicAllowance{
			SpendLimit: nil,
			Expiration: nil,
		}

		grantMsg, nerr := feegrant.NewMsgGrantAllowance(&allowance, adr, hadr)
		if nerr != nil {
			fmt.Println(nerr)
			continue
		}

		grantRes, nerr := utils.SendTx(clientCtx, cmd.Flags(), grantMsg)
		if nerr != nil {
			fmt.Println(nerr)
			continue
		}

		if grantRes.Code != 0 {
			fmt.Println(grantRes.RawLog)
			continue
		}

		fmt.Println("Done!")

	}

	fmt.Println("Finished Initialization...")
}

func (m *StrayManager) Start(cmd *cobra.Command) { // loop through stray system
	for {
		m.CollectStrays(cmd)           // query strays from the chain
		m.Distribute()                 // hands strays out to hands
		for _, hand := range m.hands { // process every stray in parallel
			go hand.Process(m.Context, m)
		}
		time.Sleep(time.Second * 20) // loop every 20 seconds
	}
}
