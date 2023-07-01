package proof

import ()

type Proof interface {
	Marshal() ([]byte, error)
	Unmarshal(raw []byte) error
}
