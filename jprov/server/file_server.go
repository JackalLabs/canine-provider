package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/strays"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/rs/cors"
	"github.com/syndtr/goleveldb/leveldb"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"

	_ "net/http/pprof"
)

func saveFile(file multipart.File, handler *multipart.FileHeader, sender string, cmd *cobra.Command, db *leveldb.DB, w *http.ResponseWriter, q *queue.UploadQueue) error {
	size := handler.Size
	ctx := utils.GetServerContextFromCmd(cmd)
	fid, merkle, _, err := utils.WriteFileToDisk(cmd, file, file, file, size, db, ctx.Logger)
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

	cidHash := sha256.New()

	var str strings.Builder // building the FID
	str.WriteString(sender)
	str.WriteString(address)
	str.WriteString(fid)

	_, err = io.WriteString(cidHash, str.String())
	if err != nil {
		return err
	}
	cid := cidHash.Sum(nil)
	strCid, err := utils.MakeCid(cid)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	msg, ctrErr := MakeContract(cmd, fid, sender, &wg, q, merkle, fmt.Sprintf("%d", size))
	if ctrErr != nil {
		ctx.Logger.Error("CONTRACT ERROR: %v", ctrErr)
		return ctrErr
	}
	wg.Wait()

	v := types.UploadResponse{
		CID: strCid,
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

	err = utils.SaveToDatabase(fid, strCid, db, ctx.Logger)
	if err != nil {
		return err
	}

	return nil
}

func MakeContract(cmd *cobra.Command, fid string, sender string, wg *sync.WaitGroup, q *queue.UploadQueue, merkleroot string, filesize string) (*types.Upload, error) {
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

	ctx.Logger.Info(fmt.Sprintf("Contract being pushed: %s", msg.String()))

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

	me, err := queryClient.Providers(context.Background(), params)
	if err != nil {
		fmt.Println("Provider not initialized on the blockchain, or connection to the RPC node has been lost. Please make sure your RPC node is available then run `jprovd init` to fix this.")
		return
	}

	providers, err := queryClient.ProvidersAll(context.Background(), &storageTypes.QueryAllProvidersRequest{})
	if err != nil {
		fmt.Println("Cannot connect to jackal blockchain.")
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
	PProfRoutes(router)

	handler := cors.Default().Handler(router)

	ctx := utils.GetServerContextFromCmd(cmd)

	threads, err := cmd.Flags().GetUint(types.FlagThreads)
	if err != nil {
		fmt.Println(err)
		return
	}

	strs, err := cmd.Flags().GetBool(types.HaltStraysFlag)
	if err != nil {
		fmt.Println(err)
		return
	}

	providerName, err := cmd.Flags().GetString(types.FlagProviderName)
	if err != nil {
		providerName = "A Storage Provider"
	}

	//fmt.Println("Testing connection...")
	//connected := testConnection(providers.Providers, me.Providers.Ip)
	//if !connected {
	//	fmt.Println("Domain not configured correctly, make sure your domain points to your provider.")
	//	return
	//}
	_ = providers
	_ = me

	manager := strays.NewStrayManager(cmd) // creating and starting the stray management system
	if !strs {
		manager.Init(cmd, threads, db)
	}
	// Start the reporting system
	reporter := InitReporter(cmd)

	go postProofs(cmd, db, &q, ctx)
	go NatCycle(cmd.Context())
	go q.StartListener(cmd, providerName)

	if !strs {
		go manager.Start(cmd)
	}

	report, err := cmd.Flags().GetBool(types.FlagDoReport)
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		for {
			if rand.Int63n(2) == 0 && report {
				err := reporter.Report(cmd)
				if err != nil {
					fmt.Println(err)
				}
			} else {
				err := reporter.AttestReport(&q)
				if err != nil {
					fmt.Println(err)
				}
			}

			time.Sleep(30 * time.Second)
		}
	}()

	port, err := cmd.Flags().GetInt(types.FlagPort)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("🌍 Started Provider: http://0.0.0.0:%d\n", port)
	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), handler)
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
