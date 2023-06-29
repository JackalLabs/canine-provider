package archive

var _ ProofArchive = &MerkleProofArchive{}// Compile time check

type MerkleProofArchive struct {}

func (m *MerkleProofArchive) Save(id []byte, p proof) error {
	return nil
}

func (m *MerkleProofArchive) Delete(id []byte) error {
	return nil
}

func (m *MerkleProofArchive) Retrieve(id []byte) (proof, error) {
	return nil, nil
}
