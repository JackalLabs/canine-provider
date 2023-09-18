package strays

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

func (m *StrayManager) AddHand(index uint) *LittleHand {
	pkeyStruct, err := crypto.ReadKey(m.ClientContext)
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
		Database:      m.Db,
		Busy:          false,
		Cmd:           m.Cmd,
		ClientContext: m.ClientContext,
		Id:            index,
		Address:       address,
	}

	m.hands = append(m.hands, &hand)
	return &hand
}

func (m *StrayManager) Distribute() { // Hand out every available stray to an idle hand
	m.Context.Logger.Debug("Distributing strays to hands...")

	for i := 0; i < len(m.hands); i++ {
		h := m.hands[i]
		m.Context.Logger.Debug(fmt.Sprintf("Distributing strays to hand #%d", h.Id))

		if h.Stray != nil { // skip all currently busy hands
			m.Context.Logger.Debug(fmt.Sprintf("Hand #%d is busy, can't give stray.", h.Id))
			continue
		}

		if len(m.Strays) == 0 { // make sure there are strays to distribute
			m.Context.Logger.Debug("There are no more strays in the pile.")
			continue
		}

		h.Stray = m.Strays[0]
		m.Strays = m.Strays[1:] // pop the first element off the queue & assign it to the hand
	}
}

func (m *StrayManager) Init() { // create all the hands for the manager
	fmt.Println("Starting initialization...")

	threads, err := m.Cmd.Flags().GetUint(types.FlagThreads)
	if err != nil {
		fmt.Println(err)
		return
	}

	var i uint
	for i = 1; i < threads+1; i++ {
		fmt.Printf("Processing stray thread %d.\n", i)
		h := m.AddHand(i)

		found := false
		for _, claimer := range m.Provider.AuthClaimers {
			if claimer == h.Address {
				found = true
				break
			}
		}
		if found {
			continue
		}

		fmt.Println("Adding hand to my claim whitelist...")

		msg := storageTypes.NewMsgAddClaimer(m.Address, h.Address)

		res, err := utils.SendTx(m.ClientContext, m.Cmd.Flags(), "", msg)
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

		adr, nerr := sdk.AccAddressFromBech32(m.Address)
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

		grantRes, nerr := utils.SendTx(m.ClientContext, m.Cmd.Flags(), "", grantMsg)
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

func (m *StrayManager) CollectStrays(cmd *cobra.Command, lastCount uint64) uint64 {
	m.Context.Logger.Info(fmt.Sprintf("Collecting strays from chain... ~ %d", lastCount))
	qClient := storageTypes.NewQueryClient(m.ClientContext)

	var val uint64
	if lastCount > 300 {
		val = uint64(m.Rand.Int63n(int64(lastCount)))
	}

	page := &query.PageRequest{
		Offset:     val,
		Limit:      300,
		Reverse:    m.Rand.Intn(2) == 0,
		CountTotal: true,
	}

	res, err := qClient.StraysAll(cmd.Context(), &storageTypes.QueryAllStraysRequest{
		Pagination: page,
	})
	if err != nil {
		m.Context.Logger.Error(err.Error())
		return 0
	}

	s := res.Strays

	if len(s) == 0 { // If there are no strays, the network has claimed them all. We will try again later.
		m.Context.Logger.Info("No strays found.")
		return 0
	}

	m.Rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })

	m.Strays = make([]*storageTypes.Strays, 0)

	for _, newStray := range s { // Only add new strays to the queue

		k := newStray
		m.Strays = append(m.Strays, &k)

	}

	return res.Pagination.Total
}

func (m *StrayManager) Start(cmd *cobra.Command) { // loop through stray system
	tm, err := cmd.Flags().GetInt64(types.FlagStrayInterval)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	m.Rand = r
	if err != nil {
		panic(err)
	}

	var s uint64
	for {
		s = m.CollectStrays(cmd, s)    // query strays from the chain
		m.Distribute()                 // hands strays out to hands
		for _, hand := range m.hands { // process every stray in parallel
			go hand.Process(m.Context, m)
		}
		time.Sleep(time.Second * time.Duration(tm)) // loop every 20 seconds
	}
}
