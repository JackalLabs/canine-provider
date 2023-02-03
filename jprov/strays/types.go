package strays

import (
	"sync"

	"github.com/jackalLabs/canine-chain/x/storage/types"
)

type StrayQueue struct {
}

type StrayManager struct {
	hands  []LittleHand
	Waiter sync.WaitGroup
	Strays []*types.Strays
}

func NewStrayManager() *StrayManager {
	return &StrayManager{
		hands:  []LittleHand{},
		Strays: []*types.Strays{},
	}
}

type LittleHand struct {
	Stray  *types.Strays
	Waiter *sync.WaitGroup
}
