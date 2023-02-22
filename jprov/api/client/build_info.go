package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JackalLabs/jackal-provider/jprov/api/types"
	provTypes "github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func GetBuildInfo(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	_ = r
	_ = ps

	port, err := cmd.Flags().GetInt(provTypes.FlagPort)
	version, err := cmd.Flags().GetString(provTypes.VersionFlag)
	noStrays, err := cmd.Flags().GetBool(provTypes.HaltStraysFlag)
	interval, err := cmd.Flags().GetUint16(provTypes.FlagInterval)
	threads, err := cmd.Flags().GetUint(provTypes.FlagThreads)
	maxMisses, err := cmd.Flags().GetInt(provTypes.FlagMaxMisses)
	chunkSize, err := cmd.Flags().GetInt64(provTypes.FlagChunkSize)
	strayInterval, err := cmd.Flags().GetInt64(provTypes.FlagStrayInterval)
	messageSize, err := cmd.Flags().GetInt(provTypes.FlagMessageSize)
	gasProof, err := cmd.Flags().GetInt(provTypes.FlagGasCap)

	if err != nil {
		fmt.Println(err)
		return
	}

	v := types.BuildResponse{
		Port:          port,
		Version:       version,
		NoStrays:      noStrays,
		Interval:      interval,
		Threads:       threads,
		MaxMisses:     maxMisses,
		ChunkSize:     chunkSize,
		StrayInterval: strayInterval,
		MessageSize:   messageSize,
		GasPerProof:   gasProof,
	}

	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}
