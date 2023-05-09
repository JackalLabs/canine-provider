package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func GetProxy(cmd *cobra.Command, w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ok := true
	var versionRes types.VersionResponse
	var versionUrl *url.URL
	var res *http.Response

	queries := r.URL.Query()
	uri := queries.Get("route")

	u, err := url.Parse(uri)
	if err != nil {
		ok = false
		goto skip
	}

	versionUrl = u.JoinPath("version")
	res, err = http.Get(versionUrl.String())
	if err != nil {
		ok = false
		goto skip
	}
	err = json.NewDecoder(res.Body).Decode(&versionRes)
	if err != nil {
		ok = false
		goto skip
	}

skip:

	okRes := types.ProxyResponse{
		Ok: ok,
	}
	err = json.NewEncoder(w).Encode(okRes)
	if err != nil {
		fmt.Println(err)
	}
}
