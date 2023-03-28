package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wealdtech/go-merkletree/sha3"

	"github.com/wealdtech/go-merkletree"
)

func main() {
	rawTree, err := os.ReadFile("benchmark/tree.json")
	if err != nil {
		panic(err)
	}

	var total int64 = 5
	var totalTime int64
	var i int64
	for ; i < total; i++ {
		t := time.Now().UnixMicro()
		for i := 0; i < 100; i++ {
			_, err := merkletree.ImportMerkleTree(rawTree, sha3.New512())
			if err != nil {
				panic(err)
			}
		}
		tt := time.Now().UnixMicro() - t
		totalTime += tt
		fmt.Printf("Took: %d microseconds.\n", tt)
	}
	totalTime = totalTime / total

	fmt.Printf("Took: %d microseconds on average.\n", totalTime)
}
