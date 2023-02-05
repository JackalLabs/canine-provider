package strays

import (
	"encoding/hex"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/cosmos/cosmos-sdk/client"
	txns "github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	storageTypes "github.com/jackalLabs/canine-chain/x/storage/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func indexPrivKey(key string, index byte) (*cryptotypes.PrivKey, error) {
	keyData, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	keyData[len(keyData)-1] += index
	k := cryptotypes.PrivKey{
		Key: keyData,
	}

	return &k, nil
}

func prepareFactory(clientCtx client.Context, txf txns.Factory) (txns.Factory, error) {
	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		return txf, err
	}

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

func (h *LittleHand) SendTx(clientCtx client.Context, flagSet *pflag.FlagSet, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	txf := txns.NewFactoryCLI(clientCtx, flagSet)

	txf, err := prepareFactory(clientCtx, txf)
	if err != nil {
		return nil, err
	}

	txf = txf.WithGas(uint64(2000000 * (len(msgs) + 1)))
	if clientCtx.Simulate {
		return nil, nil
	}

	tx, err := txns.BuildUnsignedTx(txf, msgs...)
	if err != nil {
		return nil, err
	}

	address, err := crypto.GetAddress(clientCtx)
	if err != nil {
		return nil, err
	}

	adr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, err
	}

	tx.SetFeeGranter(adr)
	err = h.Sign(txf, clientCtx, byte(h.Id), tx, true)
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

func (h *LittleHand) Sign(txf txns.Factory, clientCtx client.Context, index byte, txBuilder client.TxBuilder, overwriteSig bool) error {
	signMode := txf.SignMode()
	if signMode == signing.SignMode_SIGN_MODE_UNSPECIFIED {
		// use the SignModeHandler's default mode if unspecified
		signMode = signing.SignMode_SIGN_MODE_DIRECT
	}

	pkeyStruct, err := crypto.ReadKey(clientCtx)
	if err != nil {
		return err
	}

	key, err := indexPrivKey(pkeyStruct.Key, index)
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
			if hands.Stray == nil {
				continue
			}
			if newStray.Cid == hands.Stray.Cid {
				clean = false
			}
		}
		if clean {
			m.Strays = append(m.Strays, &newStray)
		}
	}
}
