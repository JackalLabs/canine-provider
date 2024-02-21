package server

import (

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
)

func (f *FileServer) QueryAllFilesByMerkle(fid []byte) ([]storageTypes.UnifiedFile, error) {
    req := storageTypes.QueryAllFilesByMerkle{Merkle: fid}

    resp, err := f.queryClient.AllFilesByMerkle(f.cmd.Context(), &req)
    if err != nil {
        return nil, err
    }

    return resp.Files, nil
}
