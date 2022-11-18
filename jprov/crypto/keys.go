package crypto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func KeyExists(ctx client.Context) bool {
	configPath := filepath.Join(ctx.HomeDir, "config")
	configFilePath := filepath.Join(configPath, "priv_storkey.json")

	jsonFile, err := os.Open(configFilePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return false
	}
	defer jsonFile.Close()

	return true
}

func WriteKey(ctx client.Context, key *StorPrivKey) error {
	configPath := filepath.Join(ctx.HomeDir, "config")
	configFilePath := filepath.Join(configPath, "priv_storkey.json")

	data, err := json.Marshal(key)
	if err != nil {
		return err
	}
	err = os.WriteFile(configFilePath, data, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func GetAddress(ctx client.Context) (string, error) {
	key, err := ReadKey(ctx)
	if err != nil {
		return "", err
	}

	return key.Address, nil
}

func ReadKey(ctx client.Context) (*StorPrivKey, error) {
	configPath := filepath.Join(ctx.HomeDir, "config")
	configFilePath := filepath.Join(configPath, "priv_storkey.json")

	jsonFile, err := os.Open(configFilePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := os.ReadFile(jsonFile.Name())
	if err != nil {
		return nil, err
	}

	var keyStruct StorPrivKey

	err = json.Unmarshal(byteValue, &keyStruct)
	if err != nil {
		return nil, err
	}

	return &keyStruct, nil
}

func Sign(priv *cryptotypes.PrivKey, msg []byte) ([]byte, error) {
	sig, err := priv.Sign(msg)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func ParsePrivKey(key string) (*cryptotypes.PrivKey, error) {
	keyData, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	k := cryptotypes.PrivKey{
		Key: keyData,
	}

	return &k, nil
}

func ExportPrivKey(priv *cryptotypes.PrivKey) string {
	return fmt.Sprintf("%x", priv.Key)
}
