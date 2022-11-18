package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

type DataBlock struct {
	Key   string `json:"block_name"`
	Value string `json:"block_data"`
}

type ListResponse struct {
	Files []string `json:"files"`
}

type QueueResponse struct {
	Messages []sdk.Msg `json:"messages"`
}

type DBResponse struct {
	Data []DataBlock `json:"data"`
}

type DealsResponse struct {
	Deals []storagetypes.ActiveDeals `json:"deals"`
}

type StraysResponse struct {
	Strays []storagetypes.Strays `json:"strays"`
}

type BalanceResponse struct {
	Balance *sdk.Coin `json:"balance"`
}

type SpaceResponse struct {
	Total int64 `json:"total_space"`
	Used  int64 `json:"used_space"`
	Free  int64 `json:"free_space"`
}
