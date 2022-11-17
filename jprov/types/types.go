package types

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

const MaxFileSize = 32 << 30

type IndexResponse struct {
	Status  string
	Address string
}

type UploadResponse struct {
	CID string
	FID string
}

type ErrorResponse struct {
	Error string
}

type VersionResponse struct {
	Version string
}

type ListResponse struct {
	Files []string
}

type QueueResponse struct {
	Messages []sdk.Msg
}

type Message interface{}

type Upload struct {
	Message  sdk.Msg
	Callback *sync.WaitGroup
	Err      error
	Response *sdk.TxResponse
}

type DataBlock struct {
	Key   string
	Value string
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
