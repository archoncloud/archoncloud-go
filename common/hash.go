package common

import (
	"crypto"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"hash"
	"io"
	"os"
)

const ArchonHash = crypto.SHA3_256
const ArchonHashLen = 32 // Bytes
type ArchonHashBytes [ArchonHashLen]byte
type EthAddress ecommon.Address

// Ethereum transaction
type TransactionHash [32]byte

func NewEthAddress(a []byte) *EthAddress {
	var ea EthAddress
	copy(ea[:], a)
	return &ea
}

// NewArchonHash returns the hasher used by Archon code
func NewArchonHash() hash.Hash {
	return ArchonHash.New()
}

func NewTransactionHash( txHashString string ) *TransactionHash {
	var txHash TransactionHash
	txHashSlice, err := hexutil.Decode(txHashString)
	if err == nil {
		copy(txHash[:], txHashSlice)
	}
	return &txHash
}

// HashingWriter writes streams to both a ToData destination and the ToHash hasher
type HashingWriter struct {
	ToData io.Writer
	ToHash hash.Hash
}

func NewHashingWriter(w io.Writer) *HashingWriter {
	return &HashingWriter{w, NewArchonHash()}
}

// Implement io.Writer interface
func (w HashingWriter) Write(p []byte) (n int, err error) {
	n, err = w.ToData.Write(p)
	if err == nil {
		_, err = w.ToHash.Write(p)
	}
	return n, err
}

func (w *HashingWriter) GetHash() []byte {
	return w.ToHash.Sum(nil)
}

func GetArchonHash(data []byte) []byte {
	ah := NewArchonHash()
	ah.Write(data)
	return ah.Sum(nil)
}

func GetArchonHashOf(r io.Reader) ([]byte, int64, error) {
	ah := NewArchonHash()
	n, err := io.Copy(ah, r)
	if err == nil {
		return ah.Sum(nil), n, nil
	}
	return nil, 0, err
}

func GetArchonHashOfFile(filePath string) ([]byte, int64, error) {
	fileReader, err := os.Open(filePath)
	if err == nil {
		defer fileReader.Close()
		return GetArchonHashOf(fileReader)
	}
	return nil, 0, err
}

// ArchonHashString is base58 encoded
func ArchonHashString(hash []byte) string {
	return base58.Encode(hash)
}

func ArchonHashFromString(hs string) ([]byte, error) {
	data := base58.Decode(hs)
	if len(data) != ArchonHashLen {
		return nil, fmt.Errorf("hash length is %d. should be %d", len(data), ArchonHashLen)
	}
	return data, nil
}
