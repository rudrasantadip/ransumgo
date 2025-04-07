// package main

// import (
// 	"os"

// 	"github.com/rudrasantadip/ransumgo/cli"
// )

// func main() {
// 	defer os.Exit(0)

// 	cmd := cli.CommandLine{}
// 	cmd.Run()

// }

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/rudrasantadip/ransumgo/blockchain"
	"github.com/rudrasantadip/ransumgo/network"
	"github.com/rudrasantadip/ransumgo/wallet"
)

var bc *blockchain.BlockChain

func main() {
	bc = blockchain.ContinueBlockChain("3000")

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/blocks", handleBlocks)
	http.HandleFunc("/mine", handleMine)
	http.HandleFunc("/transactions", handleTransactions)
	http.HandleFunc("/submitTx", handleSubmitTx)

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/index.html"))
	tmpl.Execute(w, nil)
}

func handleBlocks(w http.ResponseWriter, r *http.Request) {
	var blocks []map[string]interface{}

	iter := bc.Iterator()
	for {
		block := iter.Next()

		blocks = append(blocks, map[string]interface{}{
			"Hash":     fmt.Sprintf("%x", block.Hash),
			"PrevHash": fmt.Sprintf("%x", block.PrevHash),
			"Nonce":    block.Nonce,
		})

		if len(block.PrevHash) == 0 {
			break
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	mempool := network.GetMemoryPool() // create a getter for memoryPool in network package
	var txs []map[string]interface{}

	for _, tx := range mempool {
		txs = append(txs, map[string]interface{}{
			"ID": fmt.Sprintf("%x", tx.ID),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

func handleMine(w http.ResponseWriter, r *http.Request) {
	network.MineTx(bc) // assumes mineAddress is already set
	w.Write([]byte("Block mined successfully"))
}

func handleSubmitTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	from := r.FormValue("from")
	to := r.FormValue("to")
	amountStr := r.FormValue("amount")
	nodeID := r.FormValue("nodeID") // optional field from frontend
	mineNow := r.FormValue("mineNow") == "true"

	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		http.Error(w, "Invalid address", http.StatusBadRequest)
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	if nodeID == "" {
		nodeID = "3000"
	}

	bc := blockchain.ContinueBlockChain(nodeID)
	defer bc.Database.Close()

	utxoSet := blockchain.UTXOSet{Blockchain: bc}
	wallets, err := wallet.CreateWallets(nodeID)
	if err != nil {
		http.Error(w, "Unable to create/load wallets", http.StatusInternalServerError)
		return
	}

	wallet := wallets.GetWallet(from)
	tx := blockchain.NewTransaction(&wallet, to, amount, &utxoSet)

	if mineNow {
		cbTx := blockchain.CoinbaseTx(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}
		block := bc.MineBlock(txs)
		utxoSet.Update(block)
		w.Write([]byte("Block mined and transaction included!\n"))
	} else {
		network.SendTx(network.KnownNodes[0], tx)
		w.Write([]byte("Transaction submitted to network!\n"))
	}
}
