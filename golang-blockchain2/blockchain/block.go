package blockchain

type BlockChain struct {
	Blocks []*Block
}

// https://www.golangprograms.com/go-language/struct.html
type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

// to create a function that allows us to create the actual block
func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// to create a method which allows us to add a block into the blockchain
func (chain *BlockChain) AddBlock(data string) {
	prevBlock := chain.Blocks[len(chain.Blocks)-1]
	new := CreateBlock(data, prevBlock.Hash)
	chain.Blocks = append(chain.Blocks, new)
}

// to create a function which allows us to generate the Genesis Block
func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

// to create a blockchain function which will build our initial blockchain using the Genesis block
func InitBlockChain() *BlockChain {
	return &BlockChain{[]*Block{Genesis()}}
}
