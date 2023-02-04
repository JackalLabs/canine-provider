package strays

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
)

func (m *StrayManager) AddHand(db *leveldb.DB) {
	hand := LittleHand{
		Waiter:   &m.Waiter,
		Stray:    nil,
		Database: db,
		Busy:     false,
	}

	m.hands = append(m.hands, hand)
}

func (h *LittleHand) Process() { // process the stray and make the txn, when done, free the hand & delete the stray entry
	if h.Busy {
		return
	}

	h.Busy = true
	_ = "process"

	h.Stray = nil
	h.Busy = false
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

func (m *StrayManager) Init(count int, db *leveldb.DB) { // create all the hands for the manager
	for i := 0; i < count; i++ {
		m.AddHand(db)
	}
}

func (m *StrayManager) Start(cmd *cobra.Command) { // loop through stray system
	for {
		m.CollectStrays(cmd)           // query strays from the chain
		m.Distribute()                 // hands strays out to hands
		for _, hand := range m.hands { // process every stray in parallel
			go hand.Process()
		}
		time.Sleep(time.Second * 20) // loop every 20 seconds
	}
}
