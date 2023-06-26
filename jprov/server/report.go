package server

import (
	"context"
	"encoding/hex"
	"math/rand"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/JackalLabs/jackal-provider/jprov/utils"
	"github.com/cosmos/cosmos-sdk/client"
	txns "github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
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
	clientCtx := client.GetClientContextFromCmd(cmd)
	r := Reporter{
		ClientCtx: clientCtx,
		LastCount: 0,
		Context:   cmd.Context(),
	}

	return &r
}

func (r Reporter) Report(cmd *cobra.Command) error {

	pkeyStruct, err := crypto.ReadKey(r.ClientCtx)
	if err != nil {
		return nil
	}

	key, err := createPrivKey(pkeyStruct.Key)
	if err != nil {
		return nil
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		return nil
	}

	qClient := storageTypes.NewQueryClient(r.ClientCtx)

	var val uint64
	if r.LastCount > 300 {
		val = uint64(r.Rand.Int63n(r.LastCount))
	}

	page := &query.PageRequest{
		Offset:     val,
		Limit:      300,
		Reverse:    r.Rand.Intn(2) == 0,
		CountTotal: true,
	}

	qADR := storageTypes.QueryAllActiveDealsRequest{
		Pagination: page,
	}

	res, err := qClient.ActiveDealsAll(r.Context, &qADR)
	if err != nil {
		return err
	}

	deals := res.ActiveDeals

	for _, deal := range deals {

		prov := deal.Provider

		req := storageTypes.QueryProviderRequest{Address: prov}

		res, err := qClient.Providers(r.Context, &req)
		if err != nil {
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
				continue
			}

			res, err := SendTx(r.ClientCtx, cmd.Flags(), msg)
			if err != nil {
				continue
			}

			if res == nil {
				continue
			}

			if res.Code != 0 {
				continue
			}
		}

	}

	return nil
}

func (r Reporter) AttestReport(cmd *cobra.Command) error {

	pkeyStruct, err := crypto.ReadKey(r.ClientCtx)
	if err != nil {
		return nil
	}

	key, err := createPrivKey(pkeyStruct.Key)
	if err != nil {
		return nil
	}

	address, err := bech32.ConvertAndEncode(storageTypes.AddressPrefix, key.PubKey().Address().Bytes())
	if err != nil {
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
		return err
	}

	reports := res.Reports

	pKey, err := crypto.ReadKey(r.ClientCtx)
	if err != nil {
		return err
	}

	for _, report := range reports {

		attestations := report.Attestations
		for _, attest := range attestations {
			if attest.Provider == pKey.Address {
				qADR := storageTypes.QueryActiveDealRequest{
					Cid: report.Cid,
				}
				adRes, err := qClient.ActiveDeals(r.Context, &qADR)
				if err != nil {
					return err
				}

				ad := adRes.ActiveDeals

				req := storageTypes.QueryProviderRequest{Address: attest.Provider}

				providerRes, err := qClient.Providers(r.Context, &req)
				if err != nil {
					continue
				}

				ipAddress := providerRes.GetProviders().Ip

				_, err = utils.TestDownloadFileFromURL(ipAddress, ad.Fid)
				if err != nil {
					msg := storageTypes.NewMsgReport( // Creating Report
						address,
						report.Cid,
					)
					if err := msg.ValidateBasic(); err != nil {
						continue
					}

					res, err := SendTx(r.ClientCtx, cmd.Flags(), msg)
					if err != nil {
						continue
					}

					if res == nil {
						continue
					}

					if res.Code != 0 {
						continue
					}
				}

			}
		}

	}

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

	address, err := crypto.GetAddress(clientCtx)
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

	adr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, err
	}

	tx.SetFeeGranter(adr)
	err = Sign(txf, clientCtx, tx, true)
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

func Sign(txf txns.Factory, clientCtx client.Context, txBuilder client.TxBuilder, overwriteSig bool) error {
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
