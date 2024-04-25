package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
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

func buildCid(address, sender, fid string) (string, error) {
	h := sha256.New()

	var footprint strings.Builder // building FID
	footprint.WriteString(sender)
	footprint.WriteString(address)
	footprint.WriteString(fid)

	_, err := io.WriteString(h, footprint.String())
	if err != nil {
		return "", err
	}

	return utils.MakeCid(h.Sum(nil))
}
