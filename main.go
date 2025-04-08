package main

import (
	"encoding/json"
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

	http.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		files, err := os.ReadDir("./uploads")
		if err != nil {
			http.Error(w, "Failed to read uploads directory", http.StatusInternalServerError)
			return
		}

		var filenames []string
		for _, file := range files {
			if !file.IsDir() {
				filenames = append(filenames, file.Name())
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"files": filenames})
	})

	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	fmt.Println("🚀 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
