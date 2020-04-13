package interfaces

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	dht "github.com/archoncloud/archon-dht/archon"
	permLayer "github.com/archoncloud/archon-dht/permission_layer"
	"github.com/archoncloud/archoncloud-go/common"
)

type AccountType uint8

const (
	EthAccountType AccountType = iota
	NeoAccountType
)

type IAccount interface {
	GetAccountType() AccountType
	BlockchainName() string
	Permission() common.UrlPermission

	EcdsaPrivateKey() *ecdsa.PrivateKey // follows Eth standard
	PrivateKeyBytes() []byte
	PrivateKeyString() string
	PublicKeyBytes() []byte
	EcdsaPublicKeyBytes() []byte // follows Eth standard
	PublicKeyString() string
	AddressBytes() []byte
	AddressString() string

	Sign(hash []byte) (sig []byte, err error)
	Verify(hash, signature, publicKey []byte) bool

	// true if the transaction was accepted by the blockchain
	IsTxAccepted(txId string) bool

	// For client side
	GetUserName() (string, error)
	RegisterUserName(string) error

	// For Storage Provider side
	// Checks if the storage provider is registered on the blockchain
	IsSpRegistered() bool
	RegisterSP(*RegistrationInfo) (txId string, err error)
	UnregisterSP() error
	GetUploadTxInfo(txId string) (info *UploadTxInfo, err error)
	GetEarnings() (int64, error)
	NewVersionData() (*permLayer.VersionData, error)

	// Utilities
	// amount is in blockchain base (Wei/Gas)
	PrettyCurrency(amount int64) string
	// in blockchain base
	HundredthOfCent() int64
}

type EthereumValues struct {
	WeiPerByte float64
	EthStake   float64
}

type NeoValues struct {
	GasPerGigaByte float64
}

// Info needed by the SP when it received an upload request
type UploadTxInfo struct {
	TxId              string
	UserName          string
	PublicKey         []byte
	FileContainerType uint8
	Signature         []byte
	SPs               [][]byte
}

func (u *UploadTxInfo) ToJsonString() string {
	jsonData, err := json.MarshalIndent(u, "", "    ")
	if err != nil {
		return ""
	}
	return string(jsonData)
}

type RegistrationInfo struct {
	CountryA3        string
	PledgedGigaBytes float64
	Ethereum         EthereumValues
	Neo              NeoValues
	Version          int
}

func WaitForRegUnreg(acc IAccount, isReg bool, timeout time.Duration) error {
	fmt.Println("Waiting for transaction to complete...")
	start := time.Now()
	for time.Since(start) < timeout {
		isRegistered := acc.IsSpRegistered()
		if isRegistered == isReg {
			if isReg {
				common.LogInfo.Println("Register succeeded")
			} else {
				common.LogInfo.Println("Unregister succeeded")
			}
			return nil
		}
		time.Sleep(8 * time.Second)
	}
	return fmt.Errorf("registration timed out. Blockchain is busy. Try again later")
}

func GetSeed(acc IAccount) int64 {
	hash := common.GetArchonHash(acc.PrivateKeyBytes())
	seed, _ := binary.Varint(hash)
	return seed
}

func GetNodeId(acc IAccount) (nodeId string, err error) {
	nId, err := dht.GetNodeID(GetSeed(acc))
	if err != nil {
		return
	}
	nodeId = nId.Pretty()
	return
}
