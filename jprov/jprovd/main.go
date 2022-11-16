package main

import (
	"os"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, types.DefaultAppHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
