package proof

import (
	"crypto/sha256"
	"fmt"
	"io"

	merkletree "github.com/wealdtech/go-merkletree"
	"github.com/wealdtech/go-merkletree/sha3"
)

var _ Proof = &MerkleProof{}

type MerkleProof struct {
	Tree *merkletree.MerkleTree
}

func NewMerkleProof(data [][]byte) (MerkleProof, error) {
	m := MerkleProof{}
	for i, d := range data {
		h, err := hash(d, i)
		if err != nil {
			return m, err
		}
		
		data[i] = h
	}
	tree, err := merkletree.NewUsing(data, m.HashType(), false)
	m.Tree = tree
	return m, err
}

func (m *MerkleProof) Proof(data []byte, index int) (*merkletree.Proof, error) {
	d, err := hash(data, index)
	if err != nil {
		return nil, err
	}
	return m.Tree.GenerateProof(d, 0)
}

func (m *MerkleProof) Marshal() ([]byte, error) {
	return m.Tree.Export()
}

func (m *MerkleProof) Unmarshal(raw []byte) error {
	tree, err := merkletree.ImportMerkleTree(raw, m.HashType())
	m.Tree = tree
	return err
}

func (m *MerkleProof) HashType() merkletree.HashType {
	return sha3.New512()
}

func hash(data []byte, index int) ([]byte, error) {
	h := sha256.New()
	_, err := io.WriteString(h, fmt.Sprintf("%d%x", index, data))
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
