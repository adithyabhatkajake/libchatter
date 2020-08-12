package chain

import "github.com/adithyabhatkajake/libe2c/crypto"

// NewChain returns an empty chain
func NewChain() *BlockChain {
	c := &BlockChain{}
	genesis := GetGenesis()

	c.Chain = make(map[crypto.Hash]*Block)
	c.HeightBlockMap = make(map[uint64]*Block)
	c.UnconfirmedBlocks = make(map[crypto.Hash]*Block)

	// Set genesis block as the first block
	c.HeightBlockMap[genesis.Data.Index] = genesis
	c.Chain[genesis.GetHash()] = genesis

	return c
}

// Commit takes a block and commits to it
func (c *BlockChain) Commit(b *Block) error {
	c.ChainLock.Lock()
	c.Chain[b.GetBlockHash()]
	c.ChainLock.Unlock()
	return nil
}