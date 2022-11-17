package utils

import "github.com/cosmos/cosmos-sdk/types/bech32"

func MakeFid(data []byte) (string, error) {
	return bech32.ConvertAndEncode("jklf", data)
}

func MakeCid(data []byte) (string, error) {
	return bech32.ConvertAndEncode("jklc", data)
}
