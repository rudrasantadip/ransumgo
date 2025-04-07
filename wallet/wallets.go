package wallet

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const walletFile = "./tmp/wallets_%s.data"

// Wallets manages a map of address to Wallet
type Wallets struct {
	Wallets map[string]*Wallet
}

// CreateWallets loads or creates a new Wallets set
func CreateWallets(nodeId string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFile(nodeId)
	return &wallets, err
}

// AddWallet creates and stores a new Wallet
func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())
	ws.Wallets[address] = wallet
	return address
}

// GetAllAddresses returns all wallet addresses
func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string
	for addr := range ws.Wallets {
		addresses = append(addresses, addr)
	}
	return addresses
}

// GetWallet retrieves a Wallet by address
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// LoadFile reads wallets from file
func (ws *Wallets) LoadFile(nodeId string) error {
	walletFile := fmt.Sprintf(walletFile, nodeId)

	// Ensure directory exists
	err := os.MkdirAll(filepath.Dir(walletFile), os.ModePerm)
	if err != nil {
		return err
	}

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(ws)
	if err != nil {
		return err
	}
	return nil
}

// SaveFile writes wallets to file
func (ws *Wallets) SaveFile(nodeId string) {
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeId)

	// Ensure directory exists
	err := os.MkdirAll(filepath.Dir(walletFile), os.ModePerm)
	if err != nil {
		log.Panic(err)
	}

	encoder := gob.NewEncoder(&content)
	err = encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
