package archive

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
)

var _ ProofArchive = &MerkleProofArchive{}// Compile time check

type MerkleProofArchive struct {
	baseDir string
}

func NewMerkleProofArchive (ctx client.Context) MerkleProofArchive {
	return MerkleProofArchive{baseDir: ctx.HomeDir}
}

func (m *MerkleProofArchive) Save(id []byte, p proof) error {
	f, err := os.OpenFile(m.GetProofFilePath(id), os.O_WRONLY|os.O_CREATE, 0o666)
	defer f.Close()
	if err != nil {
		return err
	}
	return nil
}

func (m *MerkleProofArchive) Delete(id []byte) error {
	return nil
}

func (m *MerkleProofArchive) Retrieve(id []byte) (proof, error) {
	return nil, nil
}

func (m *MerkleProofArchive) GetProofFilePath(id []byte) string {
	return filepath.Join(m.baseDir, fmt.Sprintf("%s.tree", string(id)))
}
