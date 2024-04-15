package strays

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

func (m *StrayManager) addClaimer(littleHand LittleHand) error {
	addClaimerMsg := storageTypes.NewMsgAddClaimer(m.Address, littleHand.Address)

	resp, err := utils.SendTx(m.ClientContext, m.Cmd.Flags(), "", addClaimerMsg)
	if err != nil {
		return fmt.Errorf("failed to add claimer: %w", err)
	}

	if resp.Code != 0 {
		return fmt.Errorf("failed to add claimer: %s", resp.RawLog)
	}

	return nil
}

func (m *StrayManager) grantAllowance(littleHand LittleHand) error {
	managerAddr, err := sdk.AccAddressFromBech32(m.Address)
	if err != nil {
		return fmt.Errorf("failed to grant allowance: %w", err)
	}

	littleHandAddr, err := sdk.AccAddressFromBech32(littleHand.Address)
	if err != nil {
		return fmt.Errorf("failed to grant allowance: %w", err)
	}

	allowance := feegrant.BasicAllowance{
		SpendLimit: nil,
		Expiration: nil,
	}

	grantMsg, err := feegrant.NewMsgGrantAllowance(&allowance, managerAddr, littleHandAddr)
	if err != nil {
		return fmt.Errorf("failed to grant allowance: %w", err)
	}

	resp, err := utils.SendTx(m.ClientContext, m.Cmd.Flags(), "", grantMsg)
	if err != nil {
		return fmt.Errorf("failed to grant allowance: %w", err)
	}

	if resp.Code != 0 {
		return fmt.Errorf("failed to grant allowance: %w", err)
	}

	return nil
}

func (m *StrayManager) authorizeHand(littleHand LittleHand) error {
	err := m.addClaimer(littleHand)
	if err != nil {
		return fmt.Errorf("failed to authorize hand: %w", err)
	}

	err = m.grantAllowance(littleHand)
	if err != nil {
		return fmt.Errorf("failed to authorize hand: %w", err)
	}

	return nil
}

func (m *StrayManager) AddHand(index uint) error {
	pkeyStruct, err := crypto.ReadKey(m.ClientContext)
	if err != nil {
		return fmt.Errorf("failed to add hand: %w", err)
	}

	key, err := indexPrivKey(pkeyStruct.Key, byte(index))
	if err != nil {
		return fmt.Errorf("failed to add hand: %w", err)
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		return fmt.Errorf("failed to add hand: %w", err)
	}

	hand := LittleHand{
		Waiter:        &m.Waiter,
		Stray:         nil,
		Database:      m.archivedb,
		Busy:          false,
		Cmd:           m.Cmd,
		ClientContext: m.ClientContext,
		Id:            index,
		Address:       address,
		Archive:       m.Archive,
		Logger:        m.Context.Logger,
	}

	found := false
	for _, claimer := range m.Provider.AuthClaimers {
		if claimer == address {
			found = true
		}
	}

	if !found {
		err = m.authorizeHand(hand)
		if err != nil {
			return fmt.Errorf("failed to add hand: %w", err)
		}
	}

	m.hands = append(m.hands, &hand)
	return nil
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

func (m *StrayManager) Init() error { // create all the hands for the manager
	fmt.Println("Starting stray manager initialization...")

	threads, err := m.Cmd.Flags().GetUint(types.FlagThreads)
	if err != nil {
		return err
	}

	var i uint
	for i = 1; i < threads+1; i++ {
		fmt.Printf("Initializing little hand no. %d\n", i)
		err := m.AddHand(i)
		if err != nil {
			fmt.Printf("failed to initialize little hand no. %d: %s\n", i, err.Error())
			fmt.Printf("proceeding without little hand no. %d\n", i)
			continue
		}

		fmt.Printf("Successfully initialized little hand no. %d\n", i)
	}

	if len(m.hands) == 0 {
		return fmt.Errorf("failed to initialize any hands")
	}

	fmt.Println("Finished Initialization...")
	return nil
}

func (m *StrayManager) CollectStrays(lastCount uint64) uint64 {
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

	res, err := qClient.StraysAll(m.Cmd.Context(), &storageTypes.QueryAllStraysRequest{
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
		_, err := m.archivedb.GetContracts(newStray.Fid)
		if errors.Is(err, archive.ErrFidNotFound) {
			m.Strays = append(m.Strays, &k)
		}
	}

	return res.Pagination.Total
}

func (m *StrayManager) Start() { // loop through stray system
	tm, err := m.Cmd.Flags().GetInt64(types.FlagStrayInterval)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	m.Rand = r
	if err != nil {
		panic(err)
	}

	var s uint64
	for {
		s = m.CollectStrays(s)         // query strays from the chain
		m.Distribute()                 // hands strays out to hands
		for _, hand := range m.hands { // process every stray in parallel
			go hand.Process(m.Context, m)
		}
		time.Sleep(time.Second * time.Duration(tm)) // loop every 20 seconds
	}
}
