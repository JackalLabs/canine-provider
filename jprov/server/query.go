package server

import (
	"strconv"

    query "github.com/cosmos/cosmos-sdk/types/query"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	verified    = "verified"
	notVerified = "not verified"
	notFound    = "not found"
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

func (f *FileServer) QueryAllActiveDeals() ([]storageTypes.ActiveDeals, error) {
    req := storageTypes.QueryAllActiveDealsRequest{
        Pagination: &query.PageRequest{CountTotal: true},
    }

    activeDeals := make([]storageTypes.ActiveDeals, 0)

    resp, err := f.queryClient.ActiveDealsAll(f.cmd.Context(), &req)
    if err != nil {
        return nil, err
    }

    activeDeals = append(activeDeals, resp.ActiveDeals...)

    for len(resp.Pagination.GetNextKey()) != 0 {
        req = storageTypes.QueryAllActiveDealsRequest{
            Pagination: &query.PageRequest{Key: resp.Pagination.GetNextKey()},
        }

        resp, err = f.queryClient.ActiveDealsAll(f.cmd.Context(), &req)
        if err != nil {
            return activeDeals, err
        }
        activeDeals = append(activeDeals, resp.ActiveDeals...)
    }

    return activeDeals, nil
}


func filterMyActiveDeals(activeDeals []storageTypes.ActiveDeals, provider string) []storageTypes.ActiveDeals {
    if activeDeals == nil {
        return nil
    }

    res := make([]storageTypes.ActiveDeals, 0)

    for _, a := range(activeDeals) {
        if a.Provider == provider {
            res = append(res, a)
        }
    }
    return res
}

func (f *FileServer) QueryMyActiveDeals() ([]storageTypes.ActiveDeals, error) {
    activeDeals, err := f.QueryAllActiveDeals()
    if err != nil {
        return nil, err
    }

    return filterMyActiveDeals(activeDeals, f.provider.Address), nil
}
