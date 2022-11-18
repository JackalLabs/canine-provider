package types

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

type Message interface{}

type Upload struct {
	Message  sdk.Msg
	Callback *sync.WaitGroup
	Err      error
	Response *sdk.TxResponse
}
