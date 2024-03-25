package utils

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

type QueryService struct {
	queryClient types.QueryClient
}

func NewQueryService(cmd *cobra.Command) *QueryService {
	return &QueryService{queryClient: types.NewQueryClient(client.GetClientContextFromCmd(cmd))}
}

func (q *QueryService) QueryProvider(ctx context.Context, address string) (provider types.Providers, err error) {
	resp, err := q.queryClient.Providers(ctx, &types.QueryProviderRequest{Address: address})
	if err != nil {
		return
	}
	return resp.Providers, nil
}
