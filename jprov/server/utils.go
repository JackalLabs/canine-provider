package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/cosmos/cosmos-sdk/client"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

const ErrNotYours = "not your deal"

//nolint:all
func testConnection(providers []storageTypes.Providers, ip string) bool {
	onlineProviders := 0
	respondingProvider := 0
	outdatedProvider := 0
	checked := 0

	for _, provider := range providers {
		if onlineProviders > 20 {
			continue
		}
		checked++
		fmt.Printf("Checked with %d other providers...\n", checked)
		u, err := url.Parse(provider.Ip)
		if err != nil {
			continue
		}
		versionUrl := u.JoinPath("version")
		r, err := http.Get(versionUrl.String())
		if err != nil {
			continue
		}
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
			outdatedProvider++
			continue
		}

		respondingProvider++
	}
	fmt.Printf("Total: %d | Online: %d | Outdated: %d| Responsive: %d\n", checked, onlineProviders, outdatedProvider, respondingProvider)

	if respondingProvider < 2 && (onlineProviders-outdatedProvider) < 3 {
		return true
	}

	if respondingProvider < 2 {
		return false
	}

	return true
}

func queryBlock(clientCtx *client.Context, cid string) (int64, error) {
	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryActiveDealRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return 0, err
	}

	return res.ActiveDeals.BlockToProve, nil
}

func checkVerified(clientCtx *client.Context, cid string, self string) (bool, int64, error) {
	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryActiveDealRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return false, 0, err
	}

	if res.ActiveDeals.Provider != self {
		return false, res.ActiveDeals.DealVersion, fmt.Errorf(ErrNotYours)
	}

	return res.ActiveDeals.ProofVerified, res.ActiveDeals.DealVersion, nil
}
