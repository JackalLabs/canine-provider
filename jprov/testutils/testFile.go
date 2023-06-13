package testutils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	storagetypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

// File with a single block
type MerkleFile struct {
	data []byte
	tree merkletree.MerkleTree
}

func NewFile(data []byte) MerkleFile {
	h := sha256.New()
	_, err := io.WriteString(h, fmt.Sprintf("%d%x", 0, data))
	if err != nil {
		panic(err)
	}
	
	raw := [][]byte{h.Sum(nil)}

	tree, err := merkletree.NewUsing(raw, sha3.New512(), false)
	if err != nil {
		panic(err)
	}

	return MerkleFile{data: h.Sum(nil), tree: *tree}
}

func (m *MerkleFile) GetProof() merkletree.Proof {
	proof, err := m.tree.GenerateProof(m.data, 0)
	if err != nil {
		panic(err)
	}
	return *proof
}

func (m *MerkleFile) GetJsonProof() []byte {
	proof, err := json.Marshal(m.GetProof())
	if err != nil {
		panic(err)
	}
	return proof
}

func (m *MerkleFile) GenerateActiveDeal() storagetypes.ActiveDeals {
	return storagetypes.ActiveDeals{Blocktoprove: "0", Merkle: hex.EncodeToString(m.tree.Root())}	
} 
