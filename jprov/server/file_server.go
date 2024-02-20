package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/rs/cors"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"

	_ "net/http/pprof"
)

type FileServer struct {
	cmd *cobra.Command
	cosmosCtx client.Context
	serverCtx *utils.Context
	queryClient storageTypes.QueryClient
	archive archive.Archive
	archivedb archive.ArchiveDB
	downtimedb *archive.DowntimeDB
	provider storageTypes.Providers
	blockSize int64
	queue *queue.UploadQueue
	logger tmlog.Logger
}

func NewFileServer (
    cmd *cobra.Command,
    archivedb archive.ArchiveDB,
    downtimedb *archive.DowntimeDB,
) (fs *FileServer, err error) {
	sCtx := utils.GetServerContextFromCmd(cmd)
	clientCtx := client.GetClientContextFromCmd(cmd)
    
	blockSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		return nil, err
	}

	queue := queue.New()

	return &FileServer{
		cmd: cmd,
		cosmosCtx: clientCtx,
		serverCtx: sCtx,
		archive: archive.NewSingleCellArchive(sCtx.Config.RootDir),
        archivedb: archivedb,
        downtimedb: downtimedb,
		blockSize: blockSize,
		queryClient: storageTypes.NewQueryClient(clientCtx),
		queue: &queue,
		logger: sCtx.Logger,
	}, nil
}

func (f *FileServer) handleUploadRequest(file multipart.File, handler *multipart.FileHeader, uploader string, w *http.ResponseWriter) error {

	// Create merkle and save to disk
	tree, err := utils.CreateMerkleTree(f.blockSize, handler.Size, file, file)
	if err != nil {
		return err
	}

    fid := hex.EncodeToString(tree.Root())

	err = f.archive.WriteTreeToDisk(fid, tree)
	if err != nil {
		return err
	}

	// Save file to disk
	_, err = f.archive.WriteFileToDisk(file, fid)
	if err != nil {
		f.serverCtx.Logger.Error("handleUploadRequest: Write To Disk Error: ", err)
		return errors.New("failed to save file on server")
	}

    

	if err = writeResponse(*w, fid); err != nil {
		f.serverCtx.Logger.Error("Json Encode Error: ", err)
		return err
	}

	return nil
}

func (f *FileServer) saveToDatabase(fid string, cid string) error {
    err := f.downtimedb.Set(cid, 0) 
    if err != nil {
        return err
    }
    return f.archivedb.SetContract(cid, fid)
}

func (f *FileServer) isFileOnChain(fid string) bool {

}

func writeResponse(w http.ResponseWriter, fid string) error {
	resp := types.UploadResponse{
		CID: fid,
		FID: fid,
	}

	return json.NewEncoder(w).Encode(resp)
}

func (f *FileServer) MakeContract(fid string, sender string, wg *sync.WaitGroup, filesize string) (*types.Upload, error) {
	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		f.serverCtx.Logger.Error(err.Error())
		return nil, err
	}

	msg := storageTypes.NewMsgPostContract(
		address,
		sender,
		filesize,
		fid,
		fid,
	)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	f.serverCtx.Logger.Info(fmt.Sprintf("Contract being pushed: %s", msg.String()))

	u := types.Upload{
		Message:  msg,
		Callback: wg,
		Err:      nil,
		Response: nil,
	}

	k := &u

	f.queue.Queue = append(f.queue.Queue, k)

	return k, nil
}

func (f *FileServer) Init() (router *httprouter.Router, err error) {
	address, err := crypto.GetAddress(f.cosmosCtx)
	if err != nil {
		return
	}

	request := &storageTypes.QueryProviderRequest{
		Address: address,
	}

	response, err := f.queryClient.Providers(context.Background(), request)
	if err != nil {
		err = fmt.Errorf("Provider not initialized on the blockchain, or connection to the RPC node has been lost. Please make sure your RPC node is available then run `jprovd init` to fix this.")
		return
	}
	
	f.provider = response.Providers

	router = httprouter.New()

	f.GetRoutes(router)
	f.PostRoutes(router)
	PProfRoutes(router)

	return
}

func (f *FileServer) StartFileServer(cmd *cobra.Command) {
	router, err := f.Init()
	if err != nil {
		fmt.Println(err)
		return
	}
	handler := cors.Default().Handler(router)

	providerName, err := cmd.Flags().GetString(types.FlagProviderName)
	if err != nil {
		providerName = "A Storage Provider"
	}

	interval, err := cmd.Flags().GetUint16(types.FlagInterval)
	if err != nil {
		interval = 0
	}

	go f.StartProofServer(interval)
	go NatCycle(cmd.Context())
	go f.queue.StartListener(cmd, providerName)

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
        return
	}
}
