package crypto_test

import (
	"fmt"
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/crypto"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	require := require.New(t)

	keyString := "b7ad3d27faef9bad601c18430ca11a523d163c5071797c8bef100baba8c37737"
	data := []byte("this is a message to sign")

	key, err := crypto.ParsePrivKey(keyString)
	require.NoError(err)

	sig, err := crypto.Sign(key, data)
	require.NoError(err)

	hexSig := "818983504460f291487b103f1f5e8a960b40530a0cf235e7471d1bfe40d49fbf62c5fdfbe64fd987a68018c9ff2ccdc36b3add5715b461f1a9d8867a9afecfe1"

	require.Equal(hexSig, fmt.Sprintf("%x", sig))
}
