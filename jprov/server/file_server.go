package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/archive"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/rs/cors"

	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"

	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"

	_ "net/http/pprof"
)

type FileServer struct {
	config      utils.Config
	cmd         *cobra.Command
	serverCtx   *serverContext
	queryClient storageTypes.QueryClient
	archive     archive.Archive
	archivedb   archive.ArchiveDB
	downtimedb  *archive.DowntimeDB
	provider    storageTypes.Providers
	blockSize   int64
	queue       *queue.UploadQueue
	logger      *slog.Logger
}

func NewFileServer(
	cmd *cobra.Command,
	serverCtx utils.Context,
	archivedb archive.ArchiveDB,
	downtimedb *archive.DowntimeDB,
) (fs *FileServer, err error) {
	sCtx := utils.GetServerContextFromCmd(cmd)
	clientCtx := client.GetClientContextFromCmd(cmd)

	blockSize, err := cmd.Flags().GetInt64(types.FlagChunkSize)
	if err != nil {
		return nil, err
	}

	srvrCtx, err := newServerContext(cmd)
	if err != nil {
		return nil, err
	}

	queue := queue.New()

	return &FileServer{
		cmd:         cmd,
		serverCtx:   srvrCtx,
		archive:     archive.NewSingleCellArchive(sCtx.Config.RootDir),
		archivedb:   archivedb,
		downtimedb:  downtimedb,
		blockSize:   blockSize,
		queryClient: storageTypes.NewQueryClient(clientCtx),
		queue:       &queue,
		logger:      serverCtx.Logger,
	}, nil
}

func newServerLogger(config config) (*slog.Logger, error) {
	var handler slog.Handler

	options := slog.HandlerOptions{
		Level: config.logLevel,
	}

	if format := strings.ToUpper(config.logFormat); format == "JSON" {
		handler = slog.NewJSONHandler(os.Stdout, &options)
	} else if format == "TEXT" {
		handler = slog.NewTextHandler(os.Stdout, &options)
	} else {
		return nil, errors.New("unknown log format")
	}

	return newCtxLogger(handler), nil
}

func (f *FileServer) saveFile(file multipart.File, handler *multipart.FileHeader, sender string, w *http.ResponseWriter) error {
	fid, err := utils.MakeFID(file, file)
	if err != nil {
		return err
	}

	tree, err := utils.CreateMerkleTree(f.blockSize, handler.Size, file, file)
	if err != nil {
		return err
	}

	err = f.archive.WriteTreeToDisk(fid, tree)
	if err != nil {
		return err
	}

	_, err = f.archive.WriteFileToDisk(file, fid)
	if err != nil {
		f.logger.Error("saveFile: Write To Disk Error: ", err)
		return err
	}

	cid, err := buildCid(f.serverCtx.address, sender, fid)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	msg, ctrErr := f.MakeContract(fid, sender, &wg, string(tree.Root()), fmt.Sprintf("%d", handler.Size))
	if ctrErr != nil {
		f.logger.Error("saveFile: CONTRACT ERROR: ", ctrErr)
		return ctrErr
	}
	wg.Wait()

	if msg.Err != nil {
		f.logger.Error(msg.Err.Error())
	}

	if err = writeResponse(*w, *msg, fid, cid); err != nil {
		f.logger.Error("Json Encode Error: ", err)
		return err
	}

	err = f.saveToDatabase(fid, cid)
	if err != nil {
		return err
	}
	f.logger.Info(fmt.Sprintf("%s %s", fid, "Added to database"))

	return nil
}

func (f *FileServer) saveToDatabase(fid string, cid string) error {
	err := f.downtimedb.Set(cid, 0)
	if err != nil {
		return err
	}
	return f.archivedb.SetContract(cid, fid)
}

func writeResponse(w http.ResponseWriter, upload types.Upload, fid, cid string) error {
	if upload.Err != nil {
		resp := types.ErrorResponse{
			Error: upload.Err.Error(),
		}
		return json.NewEncoder(w).Encode(resp)
	}

	if len(cid) == 0 {
		e := fmt.Errorf("cid: '%s' is empty", cid)
		resp := types.ErrorResponse{
			Error: e.Error(),
		}
		return json.NewEncoder(w).Encode(resp)
	}

	if len(fid) == 0 {
		e := fmt.Errorf("file with cid '%s' has empty fid: '%s'", cid, fid)
		resp := types.ErrorResponse{
			Error: e.Error(),
		}
		return json.NewEncoder(w).Encode(resp)
	}

	resp := types.UploadResponse{
		CID: cid,
		FID: fid,
	}

	return json.NewEncoder(w).Encode(resp)
}

func (f *FileServer) MakeContract(fid string, sender string, wg *sync.WaitGroup, merkleroot string, filesize string) (*types.Upload, error) {
	xRoot := hex.EncodeToString([]byte(merkleroot))

	msg := storageTypes.NewMsgPostContract(
		f.serverCtx.address,
		sender,
		filesize,
		fid,
		xRoot,
	)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	f.logger.Info(fmt.Sprintf("Contract being pushed: %s", msg.String()))

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

func (f *FileServer) Init() error {
	request := &storageTypes.QueryProviderRequest{
		Address: f.serverCtx.address,
	}

	response, err := f.queryClient.Providers(context.Background(), request)
	if err != nil {
		err = fmt.Errorf("Provider not initialized on the blockchain, or connection to the RPC node has been lost. Please make sure your RPC node is available then run `jprovd init` to fix this.")
		return err
	}

	f.provider = response.Providers

	return err
}

func (f *FileServer) RecollectActiveDeals() error {
	queryActiveDeals, err := f.QueryOnlyMyActiveDeals()
	if err != nil {
		return err
	}

	for _, q := range queryActiveDeals {
		err := f.archivedb.SetContract(q.Cid, q.Fid)
		if err != nil && !errors.Is(err, archive.ErrContractAlreadyExists) {
			return err
		}
	}

	return nil
}

func (f *FileServer) StartFileServer(cmd *cobra.Command) {
	defer func() {
		log.Printf("Closing database...\n")
		err := f.archivedb.Close()
		err = errors.Join(err, f.downtimedb.Close())
		if err != nil {
			log.Fatalf("Failed to close db: %s", err)
		}
	}()
	router := httprouter.New()

	f.GetRoutes(router)
	f.PostRoutes(router)
	PProfRoutes(router)
	handler := cors.Default().Handler(router)

	providerName, err := cmd.Flags().GetString(types.FlagProviderName)
	if err != nil {
		providerName = "A Storage Provider"
	}

	interval, err := cmd.Flags().GetUint16(types.FlagInterval)
	if err != nil {
		interval = 0
	}
	// Start the reporting system
	reporter := InitReporter(cmd)

	f.logger.Info("recollecting active deals...")
	err = f.RecollectActiveDeals()
	if err != nil {
		f.logger.Error("failed to recollect lost active deals to database :", err.Error())
	}
	go f.StartProofServer(interval)
	go NatCycle(cmd.Context())
	go f.queue.StartListener(cmd, providerName)

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
				err := reporter.AttestReport(f.queue)
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

	server := http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: handler,
	}

	go func() {
		fmt.Printf("🌍 Started Provider: http://0.0.0.0:%d\n", port)
		err = server.ListenAndServe()

		if errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Storage Provider Closed\n")
			return
		} else if err != nil {
			fmt.Printf("error starting server: %s\n", err)
			return
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Printf("Signal captured, shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}
