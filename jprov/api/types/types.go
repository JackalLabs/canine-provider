package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	storagetypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

type DataBlock struct {
	Key   string `json:"block_name"`
	Value string `json:"block_data"`
}

type DowntimeBlock struct {
	CID      string `json:"cid"`
	Downtime int    `json:"downtime"`
}

type FidBlock struct {
	CID string `json:"cid"`
	FID string `json:"fid"`
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

type DowntimeResponse struct {
	Data []DowntimeBlock `json:"data"`
}

type FidResponse struct {
	Data []FidBlock `json:"data"`
}

type DealsResponse struct {
	Deals []storagetypes.LegacyActiveDeals `json:"deals"`
}

type StraysResponse struct {
	Strays []storagetypes.Strays `json:"strays"`
}

type BalanceResponse struct {
	Balance *sdk.Coin `json:"balance"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type SpaceResponse struct {
	Total int64 `json:"total_space"`
	Used  int64 `json:"used_space"`
	Free  int64 `json:"free_space"`
}

type BuildResponse struct {
	Port          int    `json:"port"`
	Version       string `json:"version_override"`
	NoStrays      bool   `json:"disable_strays"`
	Interval      uint16 `json:"proof_interval"`
	Threads       uint   `json:"thread_count"`
	MaxMisses     int    `json:"max_proof_misses"`
	ChunkSize     int64  `json:"chunk_size"`
	StrayInterval int64  `json:"stray_interval"`
	MessageSize   int    `json:"max_message_size"`
	GasPerProof   int    `json:"gas_per_proof"`
	MaxFileSize   int    `json:"max_file_size"`
}
