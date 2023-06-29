package archive

import ()

type proof = interface{}

// The ProofArchive is responsible for reading and writing
// proof of existence of a file to the user's machine.
type ProofArchive interface {
	Save(id []byte, p proof) error
	Delete(id []byte) error
	Retrieve(id []byte) (proof, error)
}

type FileArchive interface {
	Save(name string) error
	Delete(name string) error
	Retrieve(name string) error
}
