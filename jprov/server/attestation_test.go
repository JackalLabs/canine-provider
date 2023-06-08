package server_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/server"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/stretchr/testify/require"
	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

func TestVerifyAttest(t *testing.T) {
	cases := map[string]struct{
		attest types.AttestRequest
		verified bool
		expErr bool
	}{
		"wrong proof": {
			attest: types.AttestRequest{
				Cid: "-",
				Item: "0",
			},
			verified: false,
			expErr: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T){
			var data [][]byte
			h := sha256.New()
			_, err := io.WriteString(h, fmt.Sprintf("%d%x", 0, "hello world"))
			if err != nil {
				t.Error(err)
			}

			data = append(data, h.Sum(nil))

			tree, err := merkletree.NewUsing(data, sha3.New512(), false)
			if err != nil {
				t.Error(err)
			}

			proof, err := tree.GenerateProof(data[0], 0)
			if err != nil {
				t.Error(err)
			}

			jproof, err := json.Marshal(*proof)
			if err != nil{
				t.Error(err)
			}
			c.attest.HashList = string(jproof)

			activeDeal := storagetypes.ActiveDeals{
				Blocktoprove: "0",
				Merkle: hex.EncodeToString(tree.Root()),
			}

			v, e := server.VerifyAttest(activeDeal, c.attest)

			if c.verified != v {
				t.Log("expected: ", c.verified, " got: ", v)
				t.Fail()
			}
			if !c.expErr && e != nil {
				t.Log("expect no error, got: ", e)
				t.Fail()
			}
		})
	}
}

func TestAddMsgAttest(t *testing.T) {
	cases := map[string]struct{
		address string
		cid string
		expErr bool
	}{
		"invalid_address": {
			address: "invalid_address",
			cid: "jklc1dmcul9svpv0z2uzfv30lz0kcjrpdfmmfccskt06wpy8vfqrhp4nsgvgz32",
			expErr: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T){
			q := queue.UploadQueue{}
			_, err := server.AddAttestMsg(c.address, c.cid, &q)

			if c.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
