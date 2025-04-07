package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"math/big"
	"net/http"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

// Wallet stores only serializable fields
type Wallet struct {
	PrivateKey []byte // D scalar
	PublicKey  []byte
}

// Header implements http.ResponseWriter.
func (w Wallet) Header() http.Header {
	panic("unimplemented")
}

// Write implements http.ResponseWriter.
func (w Wallet) Write([]byte) (int, error) {
	panic("unimplemented")
}

// WriteHeader implements http.ResponseWriter.
func (w Wallet) WriteHeader(statusCode int) {
	panic("unimplemented")
}

// Address generates the Base58 address from public key
func (w Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := Checksum(versionedHash)
	fullHash := append(versionedHash, checksum...)
	address := Base58Encode(fullHash)
	return address
}

// MakeWallet creates a new wallet with keys
func MakeWallet() *Wallet {
	priv, pub := NewKeyPair()
	return &Wallet{PrivateKey: priv, PublicKey: pub}
}

// NewKeyPair generates a keypair using P256 curve
func NewKeyPair() ([]byte, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	privBytes := private.D.Bytes()
	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return privBytes, pub
}

// ReconstructECDSAKey rebuilds ECDSA key from D and public key
func (w *Wallet) ReconstructECDSAKey() *ecdsa.PrivateKey {
	priv := new(ecdsa.PrivateKey)
	priv.D = new(big.Int).SetBytes(w.PrivateKey)
	priv.PublicKey.Curve = elliptic.P256()
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(w.PrivateKey)
	return priv
}

// PublicKeyHash returns RIPEMD160(SHA256(pubKey))
func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)
	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	if err != nil {
		log.Panic(err)
	}
	return hasher.Sum(nil)
}

// Checksum returns first 4 bytes of double SHA256
func Checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:checksumLength]
}

// ValidateAddress validates a given wallet address
func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))
	return bytes.Compare(actualChecksum, targetChecksum) == 0
}
