package archive

import (
    "errors"
)

var (
    ErrContractAlreadyExists = errors.New("contract already exists")
    ErrContractNotFound = errors.New("contract not found")
    ErrFidNotFound = errors.New("fid not found")
)
