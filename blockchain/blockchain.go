package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

func DBexists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}
	return true
}

func ContinueBlockChain(nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if !DBexists(path) {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(path)
	db, err := openDB(path, opts)
	Handle(err)

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		return item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
	})
	Handle(err)

	return &BlockChain{lastHash, db}
}

func InitBlockChain(address, nodeId string) *BlockChain {
	path := fmt.Sprintf(dbPath, nodeId)
	if DBexists(path) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	var lastHash []byte
	opts := badger.DefaultOptions(path)
	db, err := openDB(path, opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err
	})
	Handle(err)

	return &BlockChain{lastHash, db}
}

func (chain *BlockChain) AddBlock(block *Block) {
	err := chain.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		err := txn.Set(block.Hash, block.Serialize())
		Handle(err)

		item, err := txn.Get([]byte("lh"))
		Handle(err)
		var lastHash []byte
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		Handle(err)

		item, err = txn.Get(lastHash)
		Handle(err)
		var lastBlockData []byte
		err = item.Value(func(val []byte) error {
			lastBlockData = val
			return nil
		})
		Handle(err)

		lastBlock := Deserialize(lastBlockData)
		if block.Height > lastBlock.Height {
			err = txn.Set([]byte("lh"), block.Hash)
			Handle(err)
			chain.LastHash = block.Hash
		}
		return nil
	})
	Handle(err)
}

func (chain *BlockChain) GetBestHeight() int {
	var lastBlock Block
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		var lastHash []byte
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		Handle(err)

		item, err = txn.Get(lastHash)
		Handle(err)
		var lastBlockData []byte
		err = item.Value(func(val []byte) error {
			lastBlockData = val
			return nil
		})
		Handle(err)

		lastBlock = *Deserialize(lastBlockData)
		return nil
	})
	Handle(err)

	return lastBlock.Height
}

func (chain *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(blockHash)
		if err != nil {
			return errors.New("Block is not found")
		}
		return item.Value(func(val []byte) error {
			block = *Deserialize(val)
			return nil
		})
	})
	return block, err
}

func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	iter := chain.Iterator()
	for {
		block := iter.Next()
		blocks = append(blocks, block.Hash)
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return blocks
}

func (chain *BlockChain) MineBlock(transactions []*Transaction) *Block {
	for _, tx := range transactions {
		if !chain.VerifyTransaction(tx) {
			log.Panic("Invalid Transaction")
		}
	}

	var lastHash []byte
	var lastHeight int
	var lastBlockData []byte
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
		Handle(err)

		item, err = txn.Get(lastHash)
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastBlockData = val
			return nil
		})
		Handle(err)

		lastBlock := Deserialize(lastBlockData)
		lastHeight = lastBlock.Height
		return nil
	})
	Handle(err)

	newBlock := CreateBlock(transactions, lastHash, lastHeight+1)
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	Handle(err)
	return newBlock
}

func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return UTXO
}

func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := bc.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("Transaction does not exist")
}

func (bc *BlockChain) SignTransaction(tx *Transaction, privKeyBytes []byte) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	// Convert []byte back to ecdsa.PrivateKey
	privKey := BytesToPrivateKey(privKeyBytes)

	// Sign the transaction
	tx.Sign(*privKey, prevTXs)
}

func BytesToPrivateKey(dBytes []byte) *ecdsa.PrivateKey {
	curve := elliptic.P256()
	priv := new(ecdsa.PrivateKey)
	priv.D = new(big.Int).SetBytes(dBytes)
	priv.PublicKey.Curve = curve
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(dBytes)
	return priv
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	prevTXs := make(map[string]Transaction)
	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return tx.Verify(prevTXs)
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts = retryOpts.WithSyncWrites(true)
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	db, err := badger.Open(opts)
	if err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	}
	return db, nil
}
