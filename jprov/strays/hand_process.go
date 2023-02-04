package strays

import (
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
)

func (m *StrayManager) CollectStrays(cmd *cobra.Command) {
	qClient := storageTypes.NewQueryClient(m.ClientContext)

	res, err := qClient.StraysAll(cmd.Context(), &storageTypes.QueryAllStraysRequest{})
	if err != nil {
		m.Context.Logger.Error(err.Error())
		return
	}
	s := res.Strays

	if len(s) == 0 { // If there are no strays, the network has claimed them all. We will try again later.
		return
	}

	for _, newStray := range s { // Only add new strays to the queue
		clean := true
		for _, oldStray := range m.Strays {
			if newStray.Cid == oldStray.Cid {
				clean = false
			}
		}
		for _, hands := range m.hands { // check active processes too
			if newStray.Cid == hands.Stray.Cid {
				clean = false
			}
		}
		if clean {
			m.Strays = append(m.Strays, &newStray)
		}
	}
}
