package strays

import (
	"fmt"
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

	m.hands = append(m.hands, &hand)
	return &hand
}

func (m *StrayManager) Distribute() { // Hand out every available stray to an idle hand
	m.Context.Logger.Info("Distributing strays to hands...")

	for i := 0; i < len(m.hands); i++ {
		h := m.hands[i]
		m.Context.Logger.Info(fmt.Sprintf("Distributing strays to hand #%d", h.Id))

		if h.Stray != nil { // skip all currently busy hands
			m.Context.Logger.Info(fmt.Sprintf("Hand #%d is busy, can't give stray.", h.Id))
			continue
		}

		if len(m.Strays) == 0 { // make sure there are strays to distribute
			m.Context.Logger.Info("There are no more strays in the pile.")
			continue
		}

		h.Stray = m.Strays[0]
		m.Strays = m.Strays[1:] // pop the first element off the queue & assign it to the hand
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

	qClient := storageTypes.NewQueryClient(clientCtx) // get my address from the chain
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
