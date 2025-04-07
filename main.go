package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rudrasantadip/ransumgo/cli"
)

var commandLine = cli.CommandLine{}

func main() {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "3000" // fallback node id
		os.Setenv("NODE_ID", nodeID)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/createwallet", func(w http.ResponseWriter, r *http.Request) {
		commandLine.CreateWalletHandler(w, r, nodeID)
	})

	http.HandleFunc("/listaddresses", func(w http.ResponseWriter, r *http.Request) {
		commandLine.ListAddressesHandler(w, r, nodeID)
	})

	http.HandleFunc("/getbalance", func(w http.ResponseWriter, r *http.Request) {
		commandLine.GetBalanceHandler(w, r, nodeID)
	})

	http.HandleFunc("/createblockchain", func(w http.ResponseWriter, r *http.Request) {
		commandLine.CreateBlockchainHandler(w, r, nodeID)
	})

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		commandLine.SendHandler(w, r, nodeID)
	})

	http.HandleFunc("/printchain", func(w http.ResponseWriter, r *http.Request) {
		commandLine.PrintChainHandler(w, r, nodeID)
	})

	http.HandleFunc("/reindexutxo", func(w http.ResponseWriter, r *http.Request) {
		commandLine.ReindexUTXOHandler(w, r, nodeID)
	})

	http.HandleFunc("/startnode", func(w http.ResponseWriter, r *http.Request) {
		commandLine.StartNodeHandler(w, r)
	})

	http.HandleFunc("/uploadfile", func(w http.ResponseWriter, r *http.Request) {
		commandLine.UploadFileHandler(w, r)
	})

	fmt.Println("🚀 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
