package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/rs/cors"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"

	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/julienschmidt/httprouter"
	merkletree "github.com/wealdtech/go-merkletree"

	"github.com/spf13/cobra"
)

func saveFile(file multipart.File, handler *multipart.FileHeader, sender string, cmd *cobra.Command, db *leveldb.DB, w *http.ResponseWriter, q *queue.UploadQueue) error {
	size := handler.Size

	fid, err := utils.WriteFileToDisk(cmd, file, file, file, size)
	if err != nil {
		fmt.Printf("Write To Disk Error: %v\n", err)
		return err
	}

	clientCtx, qerr := client.GetClientTxContext(cmd)
	if qerr != nil {
		return qerr
	}

	info, ierr := clientCtx.Keyring.Key(clientCtx.From)

	if ierr != nil {
		fmt.Printf("Inforing Error: %v\n", ierr)
		return ierr
	}

	ko, err := keyring.MkAccKeyOutput(info)
	if err != nil {
		fmt.Printf("Inforing Error: %v\n", ierr)
		return err
	}

	cidhash := sha256.New()

	_, err = io.WriteString(cidhash, fmt.Sprintf("%s%s%s", sender, ko.Address, fid))
	if err != nil {
		return err
	}
	cid := cidhash.Sum(nil)
	strcid, err := utils.MakeCid(cid)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	msg, ctrerr := MakeContract(cmd, fid, sender, &wg, q)
	if ctrerr != nil {
		fmt.Printf("CONTRACT ERROR: %v\n", ctrerr)
		return ctrerr
	}
	wg.Wait()

	v := types.UploadResponse{
		CID: strcid,
		FID: fid,
	}

	if msg.Err != nil {
		fmt.Println(msg.Err)
		v := types.ErrorResponse{
			Error: msg.Err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
	} else {
		err = json.NewEncoder(*w).Encode(v)
	}

	if err != nil {
		fmt.Printf("Json Encode Error: %v\n", err)
		return err
	}

	err = utils.SaveToDatabase(fid, strcid, db)
	if err != nil {
		return err
	}

	return nil
}

func MakeContract(cmd *cobra.Command, fid string, sender string, wg *sync.WaitGroup, q *queue.UploadQueue) (*types.Upload, error) {
	merkleroot, filesize, err := HashData(cmd, fid, sender)
	if err != nil {
		return nil, err
	}

	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}

	msg := storageTypes.NewMsgPostContract(
		clientCtx.GetFromAddress().String(),
		sender,
		filesize,
		fid,
		merkleroot,
	)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	u := types.Upload{
		Message:  msg,
		Callback: wg,
		Err:      nil,
		Response: nil,
	}

	k := &u

	q.Queue = append(q.Queue, k)

	return k, nil
}

func HashData(cmd *cobra.Command, fid string, sender string) (string, string, error) {
	file, err := cmd.Flags().GetString(types.DataDir)
	if err != nil {
		return "", "", err
	}

	path := fmt.Sprintf("%s/networkfiles/%s/", file, fid)
	files, err := os.ReadDir(filepath.Clean(path))
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	size := 0
	var list [][]byte

	for i := 0; i < len(files); i++ {

		path := fmt.Sprintf("%s/networkfiles/%s/%d.jkl", file, fid, i)

		dat, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			fmt.Printf("%v\n", err)
		}

		size = size + len(dat)

		h := sha256.New()
		_, err = io.WriteString(h, fmt.Sprintf("%d%x", i, dat))
		if err != nil {
			return "", "", err
		}
		hashName := h.Sum(nil)

		list = append(list, hashName)

	}

	t, err := merkletree.New(list)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	return hex.EncodeToString(t.Root()), fmt.Sprintf("%d", size), nil
}

func queryBlock(cmd *cobra.Command, cid string) (string, error) {
	clientCtx := client.GetClientContextFromCmd(cmd)

	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryGetActiveDealsRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return "", err
	}

	return res.ActiveDeals.Blocktoprove, nil
}

func checkVerified(cmd *cobra.Command, cid string) (bool, error) {
	clientCtx := client.GetClientContextFromCmd(cmd)

	queryClient := storageTypes.NewQueryClient(clientCtx)

	argCid := cid

	params := &storageTypes.QueryGetActiveDealsRequest{
		Cid: argCid,
	}

	res, err := queryClient.ActiveDeals(context.Background(), params)
	if err != nil {
		return false, err
	}

	ver, err := strconv.ParseBool(res.ActiveDeals.Proofverified)
	if err != nil {
		return false, err
	}

	return ver, nil
}

func StartFileServer(cmd *cobra.Command) {
	clientCtx, qerr := client.GetClientTxContext(cmd)
	if qerr != nil {
		fmt.Println(qerr)
		return
	}

	files, err := cmd.Flags().GetString(types.DataDir)
	if err != nil {
		return
	}

	path, err := filepath.Abs(fmt.Sprintf("%s/contracts/contractsdb", files))
	if err != nil {
		return
	}

	db, dberr := leveldb.OpenFile(path, nil)
	if dberr != nil {
		fmt.Println(dberr)
		return
	}
	router := httprouter.New()

	q := queue.UploadQueue{
		Queue:  make([]*types.Upload, 0),
		Locked: false,
	}

	GetRoutes(cmd, router, db, &q)
	PostRoutes(cmd, router, db, &q)

	handler := cors.Default().Handler(router)

	go postProofs(cmd, db, &q)
	go q.StartListener(clientCtx, cmd)
	go q.CheckStrays(clientCtx, cmd, db)

	port, err := cmd.Flags().GetString("port")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("🌍 Started Provider: http://0.0.0.0:%s\n", port)
	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), handler)
	if err != nil {
		fmt.Println(err)
		return
	}

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Storage Provider Closed\n")
		return
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
