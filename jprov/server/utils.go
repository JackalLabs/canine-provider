package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

const ErrNotYours = "not your deal"

func testConnection(providers []storageTypes.Providers, ip string) bool {
	onlineProviders := 0
	respondingProvider := 0
	outdatedProvider := 0

	for _, provider := range providers {
		u, err := url.Parse(provider.Ip)
		if err != nil {
			continue
		}
		versionUrl := u.JoinPath("version")
		r, err := http.Get(versionUrl.String())
		var versionRes types.VersionResponse
		err = json.NewDecoder(r.Body).Decode(&versionRes)
		if err != nil {
			continue
		}
		onlineProviders++

		proxyUrl := u.JoinPath("checkme")
		vals := proxyUrl.Query()
		vals.Add("route", ip)
		proxyUrl.RawQuery = vals.Encode()
		r, err = http.Get(proxyUrl.String())
		if err != nil {
			outdatedProvider++
			continue
		}
		var proxyRes types.ProxyResponse
		err = json.NewDecoder(r.Body).Decode(&proxyRes)
		if err != nil {
			continue
		}

		respondingProvider++
	}

	if respondingProvider < 2 && (onlineProviders-outdatedProvider) < 3 {
		return true
	}

	if respondingProvider < 2 {
		return false
	}

	return true
}

func queryBlock(clientCtx *client.Context, cid string) (string, error) {
	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryActiveDealRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return "", err
	}

	return res.ActiveDeals.Blocktoprove, nil
}

func checkVerified(clientCtx *client.Context, cid string, self string) (bool, error) {
	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryActiveDealRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return false, err
	}

	ver, err := strconv.ParseBool(res.ActiveDeals.Proofverified)
	if err != nil {
		return false, err
	}

	if res.ActiveDeals.Provider != self {
		return false, fmt.Errorf(ErrNotYours)
	}

	return ver, nil
}
