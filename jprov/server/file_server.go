package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/strays"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/rs/cors"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/cosmos/cosmos-sdk/client"

	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func saveFile(file multipart.File, handler *multipart.FileHeader, sender string, cmd *cobra.Command, db *leveldb.DB, w *http.ResponseWriter, q *queue.UploadQueue) error {
	size := handler.Size
	ctx := utils.GetServerContextFromCmd(cmd)
	fid, err := utils.WriteFileToDisk(cmd, file, file, file, size, ctx.Logger)
	if err != nil {
		ctx.Logger.Error("Write To Disk Error: %v", err)
		return err
	}

	clientCtx, qerr := client.GetClientTxContext(cmd)
	if qerr != nil {
		return qerr
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return err
	}

	cidhash := sha256.New()

	_, err = io.WriteString(cidhash, fmt.Sprintf("%s%s%s", sender, address, fid))
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
		ctx.Logger.Error("CONTRACT ERROR: %v", ctrerr)
		return ctrerr
	}
	wg.Wait()

	v := types.UploadResponse{
		CID: strcid,
		FID: fid,
	}

	if msg.Err != nil {
		ctx.Logger.Error(msg.Err.Error())
		v := types.ErrorResponse{
			Error: msg.Err.Error(),
		}
		(*w).WriteHeader(http.StatusInternalServerError)
		err = json.NewEncoder(*w).Encode(v)
	} else {
		err = json.NewEncoder(*w).Encode(v)
	}

	if err != nil {
		ctx.Logger.Error("Json Encode Error: %v", err)
		return err
	}

	err = utils.SaveToDatabase(fid, strcid, db, ctx.Logger)
	if err != nil {
		return err
	}

	return nil
}

func MakeContract(cmd *cobra.Command, fid string, sender string, wg *sync.WaitGroup, q *queue.UploadQueue) (*types.Upload, error) {
	merkleroot, filesize, err := HashData(cmd, fid)
	if err != nil {
		return nil, err
	}
	ctx := utils.GetServerContextFromCmd(cmd)
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return nil, err
	}

	msg := storageTypes.NewMsgPostContract(
		address,
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

func StartFileServer(cmd *cobra.Command) {
	clientCtx, qerr := client.GetClientTxContext(cmd)
	if qerr != nil {
		fmt.Println(qerr)
		return
	}

	queryClient := storageTypes.NewQueryClient(clientCtx)

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		fmt.Println(err)
		return
	}

	params := &storageTypes.QueryProviderRequest{
		Address: address,
	}

	_, err = queryClient.Providers(context.Background(), params)
	if err != nil {
		fmt.Println("Provider not initialized on the blockchain, or connection to the RPC node has been lost. Please make sure your RPC node is available then run `jprovd init` to fix this.")
		return
	}

	path := utils.GetDataPath(clientCtx)

	db, dberr := leveldb.OpenFile(path, nil)
	if dberr != nil {
		fmt.Println(dberr)
		return
	}
	router := httprouter.New()

	q := queue.New()

	GetRoutes(cmd, router, db, &q)
	PostRoutes(cmd, router, db, &q)

	handler := cors.Default().Handler(router)

	ctx := utils.GetServerContextFromCmd(cmd)

	go postProofs(cmd, db, &q, ctx)
	go NatCycle(cmd.Context())
	go q.StartListener(cmd)

	manager := strays.NewStrayManager(cmd) // creating and starting the stray management system
	manager.Init(10, db)
	go manager.Start(cmd)

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
