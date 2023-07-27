package server_test

import (
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/server"
	"github.com/JackalLabs/jackal-provider/jprov/testutils"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/stretchr/testify/require"
)

func TestVerifyAttest(t *testing.T) {
	cases := map[string]struct {
		attest   types.AttestRequest
		verified bool
		expErr   bool
	}{
		"wrong proof": {
			attest: types.AttestRequest{
				Cid:  "-",
				Item: "0",
			},
			verified: false,
			expErr:   false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testFile := testutils.NewFile([]byte("hello world"))

			c.attest.HashList = string(testFile.GetJsonProof())

			v, e := server.VerifyAttest(testFile.GenerateActiveDeal(), c.attest)

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
	cases := map[string]struct {
		address string
		cid     string
		expErr  bool
	}{
		"invalid_address": {
			address: "invalid_address",
			cid:     "jklc1dmcul9svpv0z2uzfv30lz0kcjrpdfmmfccskt06wpy8vfqrhp4nsgvgz32",
			expErr:  true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
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
