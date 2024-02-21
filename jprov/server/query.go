package server

import (
    "strconv"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

const (
    verified = "verified"
    notVerified = "not verified"
    notFound = "not found"
)

// returns verified, notVerified or notFound
func (f *FileServer) QueryContractState(cid string) string {
    req := storageTypes.QueryActiveDealRequest{Cid: cid}
    resp, err := f.queryClient.ActiveDeals(f.cmd.Context(), &req)
    if resp == nil {
        return notFound
    }

    stat, ok := status.FromError(err)
    isDealMine := resp.ActiveDeals.Provider == f.provider.Address
    if (ok && stat.Code() == codes.NotFound) || !isDealMine {
        return notFound
    }

    v, err := strconv.ParseBool(resp.ActiveDeals.Proofverified)
    if err != nil {
        // if this happens then api must have changed in chain
        f.logger.Error("failed to parse ActiveDeals.Proofverified query resp")
        return notFound
    }

    if v {
        return verified
    }

    return notVerified
}
