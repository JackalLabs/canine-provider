package types

import (
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const MaxFileSize = 32 << 30

type IndexResponse struct {
	Status  string `json:"status"`
	Address string `json:"address"`
}

type UploadResponse struct {
	CID string `json:"cid"`
	FID string `json:"fid"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type VersionResponse struct {
	Version string `json:"version"`
}

type Message interface{}

type Upload struct {
	Message  sdk.Msg         `json:"message"`
	Callback *sync.WaitGroup `json:"callback"`
	Err      error           `json:"error"`
	Response *sdk.TxResponse `json:"response"`
}
