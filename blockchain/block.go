package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp    int64
	Hash         []byte
	Transactions []*Transaction         // Standard transactions
	FileTx       *FileUploadTransaction // Optional file transaction
	PrevHash     []byte
	Nonce        int
	Height       int
}

// Calculates the hash for blocks with or without file uploads
func (b *Block) SetHash() {
	timestamp := []byte(time.Unix(b.Timestamp, 0).String())

	var data []byte

	if b.FileTx != nil {
		data = append(data, b.FileTx.FileHash...)
		data = append(data, b.FileTx.Filename...)
	} else if len(b.Transactions) > 0 {
		data = append(data, b.HashTransactions()...)
	}

	headers := bytes.Join([][]byte{
		b.PrevHash,
		timestamp,
		data,
	}, []byte{})

	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

// Create a regular transaction block (with PoW)
func CreateBlock(txs []*Transaction, prevHash []byte, height int) *Block {
	block := &Block{
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		PrevHash:     prevHash,
		Height:       height,
	}
	pow := NewProof(block)
	nonce, hash := pow.Run()
	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// Create a file-based block
func NewFileBlock(tx *FileUploadTransaction, prevHash []byte, height int) *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		FileTx:    tx,
		PrevHash:  prevHash,
		Height:    height,
	}
	block.SetHash()
	return block
}

// Genesis block (first block in chain)
func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{}, 0)
}

// Calculate Merkle Root from standard transactions
func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Serialize())
	}
	tree := NewMerkleTree(txHashes)

	return tree.RootNode.Data
}

// Serialize block for DB
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	err := encoder.Encode(b)
	Handle(err)
	return res.Bytes()
}

// Deserialize block from DB
func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	Handle(err)
	return &block
}

// Simple error handling
func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
