package server_test

import (
	"testing"
	"io"
	"fmt"
	"crypto/sha256"

	"github.com/JackalLabs/jackal-provider/jprov/server"
//	"github.com/JackalLabs/jackal-provider/jprov/types"
//	"github.com/JackalLabs/jackal-provider/jprov/testutils"
	"github.com/stretchr/testify/require"
	merkletree "github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

func TestGenerateMerkleProof(t *testing.T) {
	cases := map[string]struct {
		index int
		item []byte
		expValid bool
		expErr bool
	}{
		"valid proof": {
			index: 0,
			item: []byte("hello"),
			expValid: true,
			expErr: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T){
			data := [][]byte{[]byte("hello"), []byte("world")}
			for i, item := range data {
				h := sha256.New()
				_, err := io.WriteString(h, fmt.Sprintf("%d%x", i, item))
				require.NoError(t, err)
				data[i] = h.Sum(nil)
			}
			tree, err := merkletree.NewUsing(data, sha3.New512(), false)
			require.NoError(t, err)

			valid, _, err := server.GenerateMerkleProof(*tree, c.index, c.item)
			require.EqualValues(t, c.expValid, valid)
			if c.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
