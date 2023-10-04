package utils_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/JackalLabs/jackal-provider/jprov/utils"

	"github.com/wealdtech/go-merkletree/sha3"
	merkletree "github.com/wealdtech/go-merkletree"
)

func TestCreateMerkleTree(t *testing.T) {
    var blockSize int64 = 1024
    file, err := os.Open("../../assets/jklstorage.png")
    if err != nil {
        t.Error("Failed to open testing file: ", err)
        return
    }
    defer func() {
        if err := file.Close(); err != nil {
            t.Error(err)
        }
    }()

    stat, err := file.Stat()
    if err != nil {
        t.Error(err)
    }

    tree, err := utils.CreateMerkleTree(blockSize, stat.Size(), file, file)
    if err != nil {
        t.Error("Failed to create merkle tree: ", err)
        return
    }

    //check merkle proof
    blocks := stat.Size()/blockSize

    for i := 0; i < int(blocks); i++ {
        h := sha256.New()
        data := make([]byte, blockSize)
        _, err := file.Read(data)
        if err != nil {
            t.Error(err)
        }
        var builder strings.Builder
        _, _ = builder.WriteString(strconv.FormatInt(int64(i), 10))
        _, _ = builder.WriteString(hex.EncodeToString(data))

        _, err = io.WriteString(h, builder.String())
        if err != nil {
            t.Errorf("Failed to create block for proof: %s", err)
            return
        }
        proof, err := tree.GenerateProof(h.Sum(nil), 0)
        if err != nil {
            t.Error(err)
            return
        }

        valid, err := merkletree.VerifyProofUsing(h.Sum(nil), false, proof, [][]byte{tree.Root()}, sha3.New512())
        if err != nil {
            t.Error(err)
            return
        }
        if !valid {
            t.Error("invalid proof generated")
        }
    }
}
