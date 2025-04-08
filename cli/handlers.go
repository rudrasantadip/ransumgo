package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rudrasantadip/ransumgo/blockchain"
	"github.com/rudrasantadip/ransumgo/network"
	"github.com/rudrasantadip/ransumgo/wallet"
)

func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var freq [256]int
	for _, b := range data {
		freq[b]++
	}

	entropy := 0.0
	dataLen := float64(len(data))
	for _, count := range freq {
		if count == 0 {
			continue
		}
		p := float64(count) / dataLen
		entropy -= p * math.Log2(p)
	}

	return entropy
}

func (cli *CommandLine) CreateWalletHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	address := wallets.AddWallet()
	wallets.SaveFile(nodeID)
	w.Write([]byte(fmt.Sprintf("New address: %s\n", address)))
}

func (cli *CommandLine) ListAddressesHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	wallets, _ := wallet.CreateWallets(nodeID)
	addresses := wallets.GetAllAddresses()
	json.NewEncoder(w).Encode(addresses)
}

func (cli *CommandLine) GetBalanceHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	address := r.URL.Query().Get("address")
	if !wallet.ValidateAddress(address) {
		http.Error(w, "Invalid address", http.StatusBadRequest)
		return
	}
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	pubKeyHash := wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	balance := 0
	for _, out := range UTXOs {
		balance += out.Value
	}
	w.Write([]byte(fmt.Sprintf("Balance of %s: %d\n", address, balance)))
}

func (cli *CommandLine) CreateBlockchainHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	address := r.URL.Query().Get("address")
	if !wallet.ValidateAddress(address) {
		http.Error(w, "Invalid address", http.StatusBadRequest)
		return
	}
	chain := blockchain.InitBlockChain(address, nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()
	w.Write([]byte("Blockchain created and UTXO set reindexed"))
}

func (cli *CommandLine) SendHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	amountStr := r.URL.Query().Get("amount")
	mine := r.URL.Query().Get("mine") == "true"

	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		http.Error(w, "Invalid address", http.StatusBadRequest)
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}

	wallets, err := wallet.CreateWallets(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	wlt := wallets.GetWallet(from)
	tx := blockchain.NewTransaction(&wlt, to, amount, &UTXOSet)

	if mine {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := chain.MineBlock(txs)
		UTXOSet.Update(block)
	} else {
		network.SendTx(network.KnownNodes[0], tx)
	}
	w.Write([]byte("Transaction sent successfully"))
}

func (cli *CommandLine) PrintChainHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	iter := chain.Iterator()
	for {
		block := iter.Next()
		fmt.Fprintf(w, "Hash: %x\n", block.Hash)
		fmt.Fprintf(w, "PrevHash: %x\n", block.PrevHash)
		pow := blockchain.NewProof(block)
		fmt.Fprintf(w, "PoW: %t\n", pow.Validate())
		for _, tx := range block.Transactions {
			fmt.Fprintf(w, "%v\n", tx)
		}
		fmt.Fprintln(w, "")
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) ReindexUTXOHandler(w http.ResponseWriter, r *http.Request, nodeID string) {
	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	UTXOSet := blockchain.UTXOSet{chain}
	UTXOSet.Reindex()
	count := UTXOSet.CountTransactions()
	w.Write([]byte(fmt.Sprintf("Reindexed! UTXO count: %d\n", count)))
}

func (cli *CommandLine) StartNodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node")
	miner := r.URL.Query().Get("miner")
	go cli.StartNode(nodeID, miner)
	w.Write([]byte(fmt.Sprintf("Started node %s\n", nodeID)))
}

func (cli *CommandLine) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		http.Error(w, "NODE_ID not set in environment", http.StatusInternalServerError)
		return
	}

	wallets, err := wallet.CreateWallets(nodeID)

	if err != nil {
		http.Error(w, "Could not load wallets", http.StatusInternalServerError)
		return
	}

	addresses := wallets.GetAllAddresses()
	if len(addresses) == 0 {
		http.Error(w, "No wallet found", http.StatusBadRequest)
		return
	}
	from := addresses[0]

	// Parse the uploaded form
	err = r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusInternalServerError)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error reading file from form", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Could not read file data", http.StatusInternalServerError)
		return
	}

	entropy := calculateEntropy(fileData)
	fmt.Println(entropy)
	if entropy >= 9.5 {
		http.Error(w, fmt.Sprintf("⚠️ File rejected due to high entropy (%.2f). Possible ransomware or encrypted content.", entropy), http.StatusBadRequest)
		return
	}

	// Create uploads folder if not exists
	uploadDir := "./uploads"
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		http.Error(w, "Could not create upload directory", http.StatusInternalServerError)
		return
	}

	// Save file with safe filename
	storagePath := filepath.Join(uploadDir, filepath.Base(handler.Filename))
	err = os.WriteFile(storagePath, fileData, 0644)
	if err != nil {
		http.Error(w, "Could not save file", http.StatusInternalServerError)
		return
	}

	// Create blockchain transaction
	tx := blockchain.NewFileUploadTransaction(from, handler.Filename, fileData, storagePath)

	// Continue blockchain instance
	bc := blockchain.ContinueBlockChain(nodeID)
	defer bc.Database.Close() // Ensure database closes after use

	err = bc.AddFileBlock(tx)
	if err != nil {
		http.Error(w, "Could not add block to blockchain", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("✅ File uploaded and recorded with hash: %s", tx.FileHash)))
}
