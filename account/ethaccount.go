package account

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/client_utils"
	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/register"
	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/wallet"
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/itsmeknt/archoncloud-go/interfaces"
	dht "github.com/itsmeknt/archoncloud-go/networking/archon-dht"
	"github.com/itsmeknt/archoncloud-go/shards"
	"github.com/pariz/gountries"
)

/*
	1,000,000 GWei = 1 Eth = $138
	1 GWei = 0.0138c
	10 GWei = 0.138c
	1 GWei = 0.0138c ~ 0.01c
	Wei to cents: divide by 100000000000
*/

// Implements IAccount for Ethereum
type EthAccount struct {
	PrivateKey      *ecdsa.PrivateKey // Public key is also available through this
	address         []byte            // Ethereum ID (20 bytes)
	privateKeyBytes []byte            // 32 bytes
	publicKeyBytes  []byte            // 64 bytes The Ethereum public key is 65 bytes. First byte is always 4
}

const (
	EthToWei                 = Quintillion
	CentsToWei               = 100000000000
)

// --------------------- IAccount start -----------------------------------------------
func (acc *EthAccount) GetAccountType() interfaces.AccountType {
	return interfaces.EthAccountType
}

func (account *EthAccount) BlockchainName() string {
	return "Ethereum"
}

func (acc *EthAccount) Permission() UrlPermission {
	return Eth
}

func (acc *EthAccount) AddressBytes() []byte {
	return acc.address
}

func (acc *EthAccount) AddressString() string {
	return BytesToString(acc.AddressBytes())
}

func (account *EthAccount) EcdsaPrivateKey() *ecdsa.PrivateKey {
	return account.PrivateKey
}

func (account *EthAccount) PrivateKeyBytes() []byte {
	return account.privateKeyBytes
}

func (account *EthAccount) PrivateKeyString() string {
	return BytesToString(account.PrivateKeyBytes())
}

func (acc *EthAccount) PublicKeyBytes() []byte {
	return acc.publicKeyBytes
}

func (acc *EthAccount) EcdsaPublicKeyBytes() []byte {
	return acc.publicKeyBytes
}

func (account *EthAccount) PublicKeyString() string {
	return BytesToString(account.PublicKeyBytes())
}

// GetUserName returns the user name, if registered, otherwise empty string
// Note SPs have no user name
func (account *EthAccount) GetUserName() (userName string, err error) {
	var a EthAddress
	copy(a[:], account.AddressBytes())
	un, err := client_utils.GetUsernameFromContract(a)
	if err != nil {
		return
	}
	// User name is padded with zeros
	ix := bytes.IndexByte(un[:], 0)
	if ix >= 0 {
		userName = string(un[:ix])
	} else {
		userName = string(un[:])
	}
	return
}

func (account *EthAccount) RegisterUserName(userName string) error {
	pars := client_utils.RegisterUsernameParams{
		Username: userName,
		Wallet:   *account.GetEthereumKeyset(),
	}
	txID, err := client_utils.RegisterUsername(&pars)
	fmt.Println("Tx ID:", txID)
	fmt.Println("Waiting for registration...")
	if err == nil {
		registered, completed := WaitForCompletion(8*time.Second, 2*time.Minute, func() (interface{}, bool) {
			ok, err := account.GetUserName()
			return ok, err != nil
		})
		if !completed {
			err = errors.New("registration timed out. Blockchain is busy. Try again later")
		} else if !registered.(bool) {
			err = errors.New("registration failed")
		}
	}
	return err
}

func (acc *EthAccount) IsSpRegistered() (isRegistered bool) {
	var a [20]byte
	copy(a[:], acc.address)
	isRegistered, _ = register.CheckIfAddressIsRegistered_byteAddress(a)
	return
}

func (acc *EthAccount) RegisterSP(r *interfaces.RegistrationInfo) (txId string, err error) {
	nodeId, err := dht.GetNodeID(interfaces.GetSeed(acc))
	if err != nil {
		return
	}
	query := gountries.New()
	country, err := query.FindCountryByAlpha(r.CountryA3)
	if err != nil {
		err = fmt.Errorf("Unknown country code: %q", r.CountryA3)
		return
	}
	par := register.SPParams{
		SLALevel:       1,
		PledgedStorage: uint64(r.PledgedGigaBytes * humanize.GByte),
		Bandwidth:      0,
		CountryCode:    country.Codes,
		MinAskPrice:    uint64(r.Ethereum.WeiPerByte * Mega),
		Stake:          uint64(r.Ethereum.EthStake * EthToWei),
		NodeID:         nodeId.Pretty(),
	}
	for i := 0; i < 32; i++ {
		par.HardwareProof[i] = byte(rand.Intn(256))
	}
	txId, err = register.RegisterSP(par)
	if err != nil {
		return
	}
	if txId == "" {
		err = fmt.Errorf("registration failed")
		return
	}

	LogInfo.Printf("Register transaction ID=%s\nNode ID=%s", txId, par.NodeID)
	err = interfaces.WaitForRegUnreg(acc, true, 3*time.Minute)
	return
}

func (acc *EthAccount) UnregisterSP() (err error) {
	par := register.SPParams{}
	txId, err := register.UnregisterSP(par)
	if err == nil {
		LogInfo.Printf("Unregister transaction ID=%s\n", txId)
		err = interfaces.WaitForRegUnreg(acc, false, 3*time.Minute)
	}
	return
}

func (acc *EthAccount) GetUploadTxInfo(txId string) (info *interfaces.UploadTxInfo, err error) {
	return GetEthereumUploadTxInfo(txId)
}

func (acc *EthAccount) IsTxAccepted(txId string) bool {
	accepted, _ := client_utils.IsTxAcceptedByBlockchain(txId)
	return accepted
}

func (acc *EthAccount) ProposeUpload(fc *shards.FileContainer, s *shards.ShardsContainer, a *ArchonUrl, sps StorageProviders, maxPayment int64) (txId string, price int64, err error) {
	var shardContainerType shards.ShardContainerType
	var erasureCodeType shards.ErasureCodeType
	var shardSize int64 // this could also be a whole file
	if s != nil {
		shardContainerType = s.GetContainerType()
		erasureCodeType = s.GetErasureCode()
		shardSize = s.GetShardNumBytes()
	} else {
		shardContainerType = shards.ShardContainerNone
		erasureCodeType = shards.ErasureCodeNone
		shardSize = fc.Size
	}
	price, err = confirmPrice(acc, shardSize, sps, maxPayment)
	if err != nil {
		return
	}

	uplPar := client_utils.UploadParams{
		ServiceDuration:    1, // months, has no meaning for now
		MinSLARequirements: 1, // has no meaning for now
		UploadPmt:          uint64(price),
		ArchonFilepath:     a.String(),
		Filesize:           uint64(fc.Size),
		Shardsize:          uint64(shardSize),
		FileContainerType:  uint8(fc.Type),
		EncryptionType:     uint8(fc.EncryptionType),
		CompressionType:    fc.CompressionType,
		ShardContainerType: uint8(shardContainerType),
		ErasureCodeType:    uint8(erasureCodeType),
		SPsToUploadTo:      sps.EthAddresses(),
		CustomField:        1,
		ContainerSignature: *NewArchonSignature(fc.Signature),
	}
	txId, err = client_utils.ProposeUpload(&uplPar)
	if err == nil {
		fmt.Printf("Upload transaction ID=%s\n", txId)
		fmt.Println("Waiting for transaction acceptance...")
		_, completed := WaitForCompletion(5*time.Second, 2*time.Minute, func() (interface{}, bool) {
			accepted, _ := client_utils.IsTxAcceptedByBlockchain(txId)
			return nil, accepted
		})
		if !completed {
			err = fmt.Errorf("transaction timed out. Blockchain is busy")
		}
	}
	return
}

func (acc *EthAccount) PrettyCurrency(amount int64) string {
	return PrettyCurrencyForAccount(acc, amount)
}

func (acc *EthAccount) HundredthOfCent() int64 {
	return CentsToWei / 100
}

func (acc *EthAccount) Sign(hash []byte) (sig []byte, err error) {
	return Sign(acc,hash)
}

func (acc *EthAccount) Verify(hash, signature, publicKey []byte) bool {
	return Verify(acc,hash,signature,publicKey)
}

// --------------------- IAccount end -----------------------------------------------

// walletPath may be relative (to exe folder, or absolute)
func NewEthAccount(walletPath string, password string) (ethAcc *EthAccount, err error) {
	path := DefaultToExecutable(walletPath)
	if !FileExists(path) {
		err = fmt.Errorf("Cannot find file %q", path)
		return
	}
	keySet, err := wallet.GetEthKeySet(path, password)
	if err != nil {
		return
	}

	var account EthAccount
	account.privateKeyBytes = keySet.PrivateKey[:]
	account.PrivateKey, err = ecrypto.ToECDSA(account.privateKeyBytes)
	if err != nil {
		return
	}

	account.address = append(account.address, keySet.Address[:]...)
	account.publicKeyBytes = append(account.publicKeyBytes, keySet.PublicKey[:]...)
	ethAcc = &account
	return
}

func (acc *EthAccount) GetEthAddress() *EthAddress {
	return NewEthAddress(acc.AddressBytes())
}

func (account *EthAccount) GetEthPrivate() *[32]byte {
	var etp [32]byte
	copy(etp[:], account.PrivateKeyBytes())
	return &etp
}

func (account *EthAccount) GetEthereumKeyset() *wallet.EthereumKeyset {
	ks := wallet.EthereumKeyset{}
	copy(ks.PrivateKey[:], account.PrivateKeyBytes())
	copy(ks.PublicKey[:], account.PublicKeyBytes())
	copy(ks.Address[:], account.AddressBytes())
	return &ks
}

func PublicFromPrivate(priv *ecdsa.PrivateKey) []byte {
	pub := ecrypto.FromECDSAPub(priv.Public().(*ecdsa.PublicKey))
	// Discard firstPublicKeyByte (always the same)
	return pub[1:]
}

func PrivateKeyFromString(hexkey string) (*ecdsa.PrivateKey, error) {
	return ecrypto.HexToECDSA(strings.TrimLeft(hexkey, "0x"))
}

// GetEthereumUploadTxInfo returns info needed to verify the transaction on the SP side
func GetEthereumUploadTxInfo(txId string) (*interfaces.UploadTxInfo, error) {
	txHash := StringToBytes(txId)
	if txHash == nil {
		return nil, fmt.Errorf("transaction hash in not a hex string")
	}
	var txHashA [32]byte
	copy(txHashA[:], txHash)
	upTx, err := client_utils.GetUploadTx(txHashA)
	if err != nil {
		return nil, err
	}
	var userName string
	un := upTx.UsernameCompressed[:]
	// User name is padded with zeros
	ix := bytes.IndexByte(un[:], 0)
	if ix >= 0 {
		userName = string(un[:ix])
	} else {
		userName = string(un[:])
	}
	ui := interfaces.UploadTxInfo{
		TxId:     txId,
		UserName: userName,
	}
	ui.PublicKey = append([]byte(nil), upTx.PublicKey[:]...)
	ui.FileContainerType = upTx.InputDeconstructed.Params.FileContainerType
	ui.Signature = append([]byte(nil), upTx.InputDeconstructed.ContainerSignature[:]...)
	for _, validUploader := range upTx.InputDeconstructed.ArchonSPs {
		sp := append([]byte(nil), validUploader[:]...)
		ui.SPs = append(ui.SPs, sp)
	}
	return &ui, nil
}

func WeiString(wei int64) string {
	f := float64(wei)
	if wei >= EthToWei {
		return humanize.CommafWithDigits(f/EthToWei, 5) + " Eth"
	}
	if wei >= Mega {
		return humanize.CommafWithDigits(f/Giga, 4) + " GWei"
	}
	return humanize.CommafWithDigits(f, 0) + " Wei"
}

func WeiPerMByteString(weiPerByte int64) string {
	return humanize.CommafWithDigits(float64(weiPerByte)/Mega, 4)
}

// GenerateNewEthWallet creates a new .json wallet file
func GenerateNewEthWallet(path, password, pKey string) (err error) {
	if pKey != "" {
		err = wallet.GenerateAndSaveEthereumWalletFromPrivateKey(pKey, path, password)
	} else {
		err = wallet.GenerateAndSaveEthereumWallet(path, password)
	}
	return
}

func GenerateNewEthKey() []byte {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	return ecrypto.FromECDSA(key)
}
