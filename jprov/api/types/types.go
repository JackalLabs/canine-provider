package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

type DataBlock struct {
	Key   string
	Value string
}

type ListResponse struct {
	Files []string
}

type QueueResponse struct {
	Messages []sdk.Msg
}

type DBResponse struct {
	Data []DataBlock
}

type DealsResponse struct {
	Deals []storagetypes.ActiveDeals
}

type StraysResponse struct {
	Strays []storagetypes.Strays
}

type BalanceResponse struct {
	Balance *sdk.Coin
}
