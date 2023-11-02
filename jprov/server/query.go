package server

import (
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
)

func (f *FileServer) isThatMine(cid string) (bool) {
    req := storageTypes.QueryActiveDealRequest{Cid: cid}
    resp, err := f.queryClient.ActiveDeals(f.cmd.Context(), &req)
    if stat, ok := status.FromError(err); ok && stat.Code() == codes.NotFound {
        return false
    }
    return resp.ActiveDeals.Provider == f.provider.Address
}
