// https://www.youtube.com/watch?v=mYlHT9bB6OE&list=PLJbE2Yu2zumCe9cO3SIyragJ8pLmVv0z9&index=18
//The above is the link to this program.

package main

import (
	"fmt"
	"strconv"

	"github.com/tensor-programming/golang-blockchain/blockchain"
)

func main() {
	chain := blockchain.InitBlockChain()

	chain.AddBlock("First Block after Genesis Block")
	chain.AddBlock("Second Block after Genesis Block")
	chain.AddBlock("Third Block after Genesis Block")

	for _, block := range chain.Blocks {

		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Data in Block: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)

		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}

// check this link for the "func" https://www.golangprograms.com/go-language/functions.html
