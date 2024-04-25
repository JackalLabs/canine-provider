package types

import (
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

type VerifiedStatus int

const (
	Verified VerifiedStatus = iota
	NotVerified
	NotFound
	Error
)

// Returns state of the active deal contract based on query response.
// Returns Error status with non-nil error if respErr is unknown code or parsing resp failed.
func ContractState(resp *storageTypes.QueryActiveDealResponse, respErr error) (VerifiedStatus, error) {
	if respErr != nil {
		stat, ok := status.FromError(respErr)
		if !ok { // unknown grpc error
			return Error, respErr
		}

		if codes.NotFound == stat.Code() {
			return NotFound, nil
		}

		return Error, respErr
	}

	verified, err := strconv.ParseBool(resp.ActiveDeals.Proofverified)
	if err != nil {
		return Error, err
	}

	if verified {
		return Verified, err
	}
	return NotVerified, err
}
