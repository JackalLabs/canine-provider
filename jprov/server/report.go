package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/queue"
	"github.com/JackalLabs/jackal-provider/jprov/types"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	txns "github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	storageTypes "github.com/jackalLabs/canine-chain/v3/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func createPrivKey(key string) (*cryptotypes.PrivKey, error) {
	keyData, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}

	reportString := "reporting"

	for i, ch := range reportString {
		keyData[len(keyData)-i-1] += byte(ch)
	}

	k := cryptotypes.PrivKey{
		Key: keyData,
	}

	return &k, nil
}

type Reporter struct {
	ClientCtx client.Context
	Context   context.Context
	LastCount int64
	Rand      *rand.Rand
}

func InitReporter(cmd *cobra.Command) *Reporter {
	fmt.Println("Initializing report system...")
	clientCtx := client.GetClientContextFromCmd(cmd)

	randy := rand.New(rand.NewSource(time.Now().UnixNano()))

	r := Reporter{
		ClientCtx: clientCtx,
		LastCount: 0,
		Context:   cmd.Context(),
		Rand:      randy,
	}

	allowance := feegrant.BasicAllowance{
		SpendLimit: nil,
		Expiration: nil,
	}

	pkeyStruct, err := crypto.ReadKey(r.ClientCtx)
	if err != nil {
		return &r
	}

	key, err := createPrivKey(pkeyStruct.Key)
	if err != nil {
		return &r
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		return &r
	}
	fmt.Printf("Creating a fee allowance for %s from %s", address, pkeyStruct.Address)

	myAddress, err := sdk.AccAddressFromBech32(pkeyStruct.Address)
	if err != nil {
		return &r
	}
	reportAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return &r
	}

	grantMsg, err := feegrant.NewMsgGrantAllowance(&allowance, myAddress, reportAddress)
	if err != nil {
		fmt.Println(err)
		return &r
	}

	grantRes, err := utils.SendTx(clientCtx, cmd.Flags(), "", grantMsg)
	if err != nil {
		return &r
	}

	if grantRes.Code != 0 {
		fmt.Println(grantRes.RawLog)
		return &r
	}

	fmt.Println("Done!")

	return &r
}

func (r Reporter) Report(cmd *cobra.Command) error {
	fmt.Println("Attempting to report bad actors...")
	defer fmt.Println("Done report!")

	pkeyStruct, err := crypto.ReadKey(r.ClientCtx)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	key, err := createPrivKey(pkeyStruct.Key)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		fmt.Println(err)
		return nil
	}

	qClient := storageTypes.NewQueryClient(r.ClientCtx)

	page := &query.PageRequest{
		Offset:     0,
		Limit:      300,
		Reverse:    r.Rand.Intn(2) == 0,
		CountTotal: true,
	}

	qADR := storageTypes.QueryAllActiveDealsRequest{
		Pagination: page,
	}

	res, err := qClient.ActiveDealsAll(r.Context, &qADR)
	if err != nil {
		fmt.Println(err)
		return err
	}

	deals := res.ActiveDeals

	for _, deal := range deals {

		prov := deal.Provider

		if prov == pkeyStruct.Address {
			continue
		}

		req := storageTypes.QueryProviderRequest{Address: prov}

		res, err := qClient.Providers(r.Context, &req)
		if err != nil {
			fmt.Println(err)
			continue
		}

		ipAddress := res.GetProviders().Ip

		_, err = utils.TestDownloadFileFromURL(ipAddress, deal.Fid)
		if err != nil {
			msg := storageTypes.NewMsgRequestReportForm( // Creating Report
				address,
				deal.Cid,
			)
			if err := msg.ValidateBasic(); err != nil {
				fmt.Println(err)
				continue
			}

			res, err := SendTx(r.ClientCtx, cmd.Flags(), msg)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if res == nil {
				fmt.Println("failed to sentTx")
				continue
			}

			fmt.Println(res.RawLog)

			if res.Code != 0 {
				fmt.Println("failed tx")
				continue
			}

			fmt.Printf("Successfully reported %s\n", deal.Cid)

			return nil
		}
	}

	return nil
}

func (r Reporter) AttestReport(queue *queue.UploadQueue) error {
	fmt.Println("Attempting to attest to reports...")

	pkeyStruct, err := crypto.ReadKey(r.ClientCtx)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	qClient := storageTypes.NewQueryClient(r.ClientCtx)

	page := &query.PageRequest{
		Offset:     0,
		Limit:      300,
		Reverse:    r.Rand.Intn(2) == 0,
		CountTotal: true,
	}

	qARR := storageTypes.QueryAllReportRequest{
		Pagination: page,
	}

	res, err := qClient.ReportsAll(r.Context, &qARR)
	if err != nil {
		fmt.Println(err)
		return err
	}

	reports := res.Reports

	if len(reports) == 0 {
		fmt.Println("no reports to attest to.")
	}

	for _, report := range reports {
		attestations := report.Attestations
		for _, attest := range attestations {
			if attest.Provider == pkeyStruct.Address {
				fmt.Printf("attempting to attest to %s...\n", report.Cid)

				if attest.Complete {
					fmt.Println("Already completed")
					continue
				}

				qADR := storageTypes.QueryActiveDealRequest{
					Cid: report.Cid,
				}
				adRes, err := qClient.ActiveDeals(r.Context, &qADR)
				if err != nil {
					fmt.Println(err)
					return err
				}

				ad := adRes.ActiveDeals

				if ad.Provider == pkeyStruct.Address {
					fmt.Println("skipping reporting myself 😅")
					continue
				}

				req := storageTypes.QueryProviderRequest{Address: ad.Provider}

				providerRes, err := qClient.Providers(r.Context, &req)
				if err != nil {
					fmt.Println(err)
					continue
				}

				ipAddress := providerRes.GetProviders().Ip

				fmt.Printf("trying to downloading file from %s...\n", ipAddress)
				_, err = utils.TestDownloadFileFromURL(ipAddress, ad.Fid)
				if err == nil {
					fmt.Println("successfully downloaded file.")
					break
				}
				fmt.Println("failed to download file.")

				msg := storageTypes.NewMsgReport( // Creating Report
					pkeyStruct.Address,
					report.Cid,
				)
				if err := msg.ValidateBasic(); err != nil {
					fmt.Println(err)
					continue
				}

				var wg sync.WaitGroup
				wg.Add(1)

				upload := types.Upload{
					Message:  msg,
					Callback: &wg,
					Err:      nil,
					Response: nil,
				}
				queue.Append(&upload)
				wg.Wait()

				if upload.Err != nil {
					fmt.Println(upload.Err)
					continue
				}

				if upload.Response == nil {
					fmt.Println("empty response from report attestation, something is wrong")
					continue
				}

				fmt.Println(upload.Response.RawLog)

				if upload.Response.Code != 0 {
					fmt.Println(err)
					continue
				}

				break

			}
		}
	}
	fmt.Println("Done attesting to reports!")

	return nil
}

func prepareFactory(clientCtx client.Context, address string, txf txns.Factory) (txns.Factory, error) {
	from, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return txf, err
	}

	if err := txf.AccountRetriever().EnsureExists(clientCtx, from); err != nil {
		return txf, err
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := txf.AccountRetriever().GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}

func SendTx(clientCtx client.Context, flagSet *pflag.FlagSet, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	txf := txns.NewFactoryCLI(clientCtx, flagSet)

	pkeyStruct, err := crypto.ReadKey(clientCtx)
	if err != nil {
		return nil, err
	}

	key, err := createPrivKey(pkeyStruct.Key)
	if err != nil {
		return nil, err
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		return nil, err
	}

	txf, err = prepareFactory(clientCtx, address, txf)
	if err != nil {
		return nil, err
	}

	if txf.SimulateAndExecute() || clientCtx.Simulate {
		_, adjusted, err := txns.CalculateGas(clientCtx, txf, msgs...)
		if err != nil {
			return nil, err
		}

		txf = txf.WithGas(adjusted)
		//_, _ = fmt.Fprintf(os.Stderr, "%s\n", txns.GasEstimateResponse{GasEstimate: txf.Gas()})
	}
	if clientCtx.Simulate {
		return nil, nil
	}

	tx, err := txns.BuildUnsignedTx(txf, msgs...)
	if err != nil {
		return nil, err
	}

	adr, err := sdk.AccAddressFromBech32(pkeyStruct.Address)
	if err != nil {
		return nil, err
	}

	tx.SetFeeGranter(adr)
	err = sign(txf, clientCtx, tx, true)
	if err != nil {
		return nil, err
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(tx.GetTx())
	if err != nil {
		return nil, err
	}

	// broadcast to a Tendermint node
	res, err := clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return nil, err
	}

	return res, err
}

func sign(txf txns.Factory, clientCtx client.Context, txBuilder client.TxBuilder, overwriteSig bool) error {
	signMode := txf.SignMode()
	if signMode == signing.SignMode_SIGN_MODE_UNSPECIFIED {
		// use the SignModeHandler's default mode if unspecified
		signMode = signing.SignMode_SIGN_MODE_DIRECT
	}

	pkeyStruct, err := crypto.ReadKey(clientCtx)
	if err != nil {
		return err
	}

	key, err := createPrivKey(pkeyStruct.Key)
	if err != nil {
		return err
	}

	pubKey := key.PubKey()
	signerData := authsigning.SignerData{
		ChainID:       txf.ChainID(),
		AccountNumber: txf.AccountNumber(),
		Sequence:      txf.Sequence(),
	}

	// For SIGN_MODE_DIRECT, calling SetSignatures calls setSignerInfos on
	// TxBuilder under the hood, and SignerInfos is needed to generated the
	// sign bytes. This is the reason for setting SetSignatures here, with a
	// nil signature.
	//
	// Note: this line is not needed for SIGN_MODE_LEGACY_AMINO, but putting it
	// also doesn't affect its generated sign bytes, so for code's simplicity
	// sake, we put it here.
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: nil,
	}
	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: txf.Sequence(),
	}
	var prevSignatures []signing.SignatureV2
	if !overwriteSig {
		prevSignatures, err = txBuilder.GetTx().GetSignaturesV2()
		if err != nil {
			return err
		}
	}
	if err := txBuilder.SetSignatures(sig); err != nil {
		return err
	}

	// Generate the bytes to be signed.
	bytesToSign, err := clientCtx.TxConfig.SignModeHandler().GetSignBytes(signMode, signerData, txBuilder.GetTx())
	if err != nil {
		return err
	}

	// Sign those bytes
	sigBytes, err := crypto.Sign(key, bytesToSign)
	if err != nil {
		return err
	}

	// Construct the SignatureV2 struct
	sigData = signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: sigBytes,
	}
	sig = signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: txf.Sequence(),
	}

	if overwriteSig {
		return txBuilder.SetSignatures(sig)
	}
	prevSignatures = append(prevSignatures, sig)
	return txBuilder.SetSignatures(prevSignatures...)
}
