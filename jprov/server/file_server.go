package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/strays"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/rs/cors"
	"github.com/syndtr/goleveldb/leveldb"

	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"

	_ "net/http/pprof"
)

const FilePerm os.FileMode = 0o666

type FileServer struct {
	blockSize int64
	Logger    log.Logger
}

func NewFileServer(blockSize int64, logger log.Logger) (*FileServer, error) {
	return &FileServer{blockSize: blockSize, Logger: logger}, nil
}

// GetPiece returns a piece of block at index of the fid file.
func (f *FileServer) GetPiece(ctx client.Context, fid string, index int64) (block []byte, err error) {
	file, err := os.Open(utils.GetContentsPath(ctx, fid))
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	block = make([]byte, f.blockSize)
	n, err := file.ReadAt(block, index*f.blockSize)
	// ignoring io.EOF with n > 0 because the file size is not always n * blockSize
	if (err != nil && err != io.EOF) || (err == io.EOF && n == 0) {
		return
	}

	return block, nil
}

// WriteToDisk creates named file with data as contents at the directory.
// The directory is created if it doesn't exist.
// * This will consume the reader and close the io
func (f *FileServer) WriteToDisk(data io.Reader, closer io.Closer, dir, name string) (written int64, err error) {
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return
	}

	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE, FilePerm)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())

		if closer != nil {
			err = errors.Join(err, file.Close())
		}
	}()

	written, err = io.Copy(file, data)
	if err != nil {
		log := fmt.Sprintf("WriteToDisk: failed to write data to disk (wrote %d bytes)", written)
		f.Logger.Error(log)
		return
	}

	return
}

func (f *FileServer) saveFile(file multipart.File, handler *multipart.FileHeader, sender string, cmd *cobra.Command, db *leveldb.DB, w *http.ResponseWriter, q *queue.UploadQueue) error {
	ctx := utils.GetServerContextFromCmd(cmd)
	clientCtx, qerr := client.GetClientTxContext(cmd)
	if qerr != nil {
		return qerr
	}

	fid, err := utils.MakeFID(file, file)
	if err != nil {
		return err
	}

	// Create merkle and save to disk
	merkle, err := utils.CreateMerkleTree(f.blockSize, handler.Size, file, file)
	if err != nil {
		return err
	}

	exportedTree, err := merkle.Export()
	if err != nil {
		return err
	}

	buffer := bytes.NewReader(exportedTree)
	_, err = f.WriteToDisk(buffer, nil, utils.GetStorageDirForTree(clientCtx), utils.GetFileNameForTree(fid))
	if err != nil {
		return err
	}

	// Save file to disk
	_, err = f.WriteToDisk(file, file, utils.GetStoragePath(clientCtx, fid), fid)
	if err != nil {
		ctx.Logger.Error("Write To Disk Error: %v", err)
		return err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		ctx.Logger.Error(err.Error())
		return err
	}

	cid, err := buildCid(address, sender, fid)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	msg, ctrErr := MakeContract(cmd, fid, sender, &wg, q, string(merkle.Root()), fmt.Sprintf("%d", handler.Size))
	if ctrErr != nil {
		ctx.Logger.Error("CONTRACT ERROR: %v", ctrErr)
		return ctrErr
	}
	wg.Wait()

	if msg.Err != nil {
		ctx.Logger.Error(msg.Err.Error())
	}

	if err = writeResponse(*w, *msg, fid, cid); err != nil {
		ctx.Logger.Error("Json Encode Error: %v", err)
		return err
	}

	err = utils.SaveToDatabase(fid, cid, db, ctx.Logger)
	if err != nil {
		return err
	}

	return nil
}

func writeResponse(w http.ResponseWriter, upload types.Upload, fid, cid string) error {
	if upload.Err != nil {
		resp := types.ErrorResponse{
			Error: upload.Err.Error(),
		}
		return json.NewEncoder(w).Encode(resp)
	}

	resp := types.UploadResponse{
		CID: cid,
		FID: fid,
	}

	return json.NewEncoder(w).Encode(resp)
}

func buildCid(address, sender, fid string) (string, error) {
	h := sha256.New()

	var footprint strings.Builder // building FID
	footprint.WriteString(sender)
	footprint.WriteString(address)
	footprint.WriteString(fid)

	_, err := io.WriteString(h, footprint.String())
	if err != nil {
		return "", err
	}

	return utils.MakeCid(h.Sum(nil))
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

	ctx := utils.GetServerContextFromCmd(cmd)

	blockSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		fmt.Println("Cannot get chunk size")
		return
	}

	fs, err := NewFileServer(blockSize, ctx.Logger)
	if err != nil {
		fmt.Println("Failed to initialize file server")
		return
	}

	GetRoutes(cmd, router, db, &q)
	PostRoutes(cmd, fs, router, db, &q)
	PProfRoutes(router)

	handler := cors.Default().Handler(router)

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

	go fs.postProofs(cmd, db, &q, ctx)
	go NatCycle(cmd.Context())
	go q.StartListener(cmd, providerName)

	if !strs {
		go manager.Start(cmd)
	}

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
