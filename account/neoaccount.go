package account

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/archoncloud/archoncloud-go/blockchainAPI/neo"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/archoncloud/archoncloud-go/shards"
	"github.com/dustin/go-humanize"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/joeqian10/neo-gogogo/helper"
	"github.com/joeqian10/neo-gogogo/wallet"
	"github.com/pkg/errors"
	"strings"
)
/*
	Feb 2020
	1 NEO = $14.882922
	1 GAS = $2.14
	The smallest unit of GAS is 0.00000001 (10^8)
	float GAS to cents: multiply by 200
	0.4c/G -> 0.002Gas/G
*/

// Implements IAccount for Neo
// private key: 32 bytes
// public key: 65 bytes (uncompressed: 0x04 + X (32 bytes) + Y (32 bytes))

type NeoAccount struct {
	neoWallet *wallet.Account
	eth *EthereumKey
}

// --------------------- IAccount start -----------------------------------------------
func (acc *NeoAccount) GetAccountType() interfaces.AccountType {
	return interfaces.NeoAccountType
}

func (account *NeoAccount) BlockchainName() string {
	return "Neo"
}

func (acc *NeoAccount) Permission() UrlPermission {
	return Neo
}

func (account *NeoAccount) AddressBytes() []byte {
	addr, _ :=  helper.AddressToScriptHash(account.neoWallet.Address)
	return addr.Bytes()
}

func (account *NeoAccount) AddressString() string {
	return account.neoWallet.Address
}

func (account *NeoAccount) PrivateKeyBytes() []byte {
	return account.neoWallet.KeyPair.PrivateKey
}

func (account *NeoAccount) PrivateKeyString() string {
	return account.neoWallet.KeyPair.String()
}

func (account *NeoAccount) PublicKeyBytes() []byte {
	return account.neoWallet.KeyPair.PublicKey.EncodeCompression()
}

func (acc *NeoAccount) EcdsaPrivateKey() *ecdsa.PrivateKey {
	epriv, _ := ecrypto.ToECDSA(acc.eth.PrivateKey)
	return epriv
}

func (acc *NeoAccount) EcdsaPublicKeyBytes() []byte {
	return acc.eth.PublicKey
}

func (account *NeoAccount) PublicKeyString() string {
	return account.neoWallet.KeyPair.PublicKey.String()
}

// GetUserName returns the user name, if registered, otherwise empty string
// Note SPs have no user name
func (acc *NeoAccount) GetUserName() (string, error) {
	return neo.GetUserName(acc.AddressString())
}

func (acc *NeoAccount) RegisterUserName(userName string) error {
	return neo.RegisterUserName(acc.neoWallet, userName)
}

func (acc *NeoAccount) PrettyCurrency(amount int64) string {
	return PrettyCurrencyForAccount(acc, amount)
}

func (acc *NeoAccount) HundredthOfCent() int64 {
	return helper.Fixed8FromFloat64(0.005).Value
}

func (acc *NeoAccount) IsSpRegistered() bool {
	return neo.IsSpRegistered(acc.AddressString())
}

func (acc *NeoAccount) RegisterSP(r *interfaces.RegistrationInfo) (txId string, err error) {
	prof, err := neo.NewNeoSpProfileFromReg(r)
	if err != nil {return}
	prof.NodeId, err = interfaces.GetNodeId(acc)
	if err != nil {return}
	txId, err = neo.RegisterSp(acc.neoWallet, prof)
	return
}

func (acc *NeoAccount) UnregisterSP() error {
	return neo.UnregisterSp(acc.neoWallet)
}

func (acc *NeoAccount) GetUploadTxInfo(txId string) (pInfo *interfaces.UploadTxInfo, err error) {
	return neo.GetUploadTxInfo(txId)
}

func (acc *NeoAccount) IsTxAccepted(txId string) bool {
	accepted, _ := neo.IsTxAccepted(txId)
	return accepted
}

func (acc *NeoAccount) ProposeUpload(fc *shards.FileContainer, s *shards.ShardsContainer, a *ArchonUrl, sps StorageProviders, maxPayment int64) (txId string, totalPrice int64, err error) {
	var shardSize int64	// this could also be a whole file
	if s != nil {
		shardSize = s.GetShardNumBytes()
	} else {
		shardSize = fc.Size
	}
	totalPrice, err = confirmPrice(acc, shardSize, sps, maxPayment)
	if err != nil {return}

	neo.MintCGasIfNeeded(acc.neoWallet, totalPrice)

	pars := neo.UploadParamsForNeo{}
	pars.UserName = a.Username
	pars.FileContainerType = int(fc.Type)
	pars.ContainerSignature = RawBytesToString(fc.Signature)
	pars.PublicKey = RawBytesToString(acc.EcdsaPublicKeyBytes())
	// totalPrice is assumed an exact multiple
	spPayment := totalPrice / int64(len(sps))
	mBytes := MegaBytes(shardSize)
	for _, sp := range sps {
		// For Neo: one transaction per SP in order to stay in free tier
		pars.SpAddress = sp.Address
		_, err = neo.ProposeUpload(acc.neoWallet, &pars, spPayment, mBytes, false)
	}
	return
}

/*
func (account *NeoAccount) Sign(hash []byte) ([]byte, error) {
	return account.neoWallet.KeyPair.Sign(hash)
}

func (acc *NeoAccount) Verify(hash, signature, publicKey []byte) bool {
	p := acc.neoWallet.KeyPair.PublicKey
	if publicKey != nil {
		p, _ = keys.NewPublicKey(publicKey)
	}
	return keys.VerifySignature(hash, signature, p)
}
*/

func (acc *NeoAccount) Sign(hash []byte) (sig []byte, err error) {
	return Sign(acc,hash)
}

func (acc *NeoAccount) Verify(hash, signature, publicKey []byte) bool {
	return Verify(acc,hash,signature,publicKey)
}

func (acc *NeoAccount) GetEarnings() (int64, error) {
	return neo.GetCGASBalanceOf(acc.AddressString())
}

// --------------------- IAccount end -----------------------------------------------

// walletPath may be relative (to exe folder) or absolute
func NewNeoAccount(walletPath string, password string) (acc *NeoAccount, err error) {
	if !FileExists(walletPath) {
		err = fmt.Errorf("file %q does not exist", walletPath)
		return
	}
	w, err := wallet.NewWalletFromFile(walletPath)
	if err != nil {return}
	err = w.DecryptAll(password)
	if err != nil {
		err = errors.Wrap(err, "wrong password")
		return
	}

	if len(w.Accounts) == 0 {
		err = fmt.Errorf("%q does not contain any accounts", walletPath)
		return
	}
	wa := w.Accounts[0]
	for _, a := range w.Accounts {
		// prefer default
		if a.Default {
			wa = a
			break
		}
	}
	eth, err := ToEcdsa(wa.KeyPair.PrivateKey)
	if err != nil {return}
	acc = &NeoAccount{wa, eth }
	return
}

func NewNeoAccountFromWif(wif string) (acc *NeoAccount, err error) {
	w, err := wallet.NewAccountFromWIF(wif)
	eth, err := ToEcdsa(w.KeyPair.PrivateKey)
	if err != nil {return}
	acc = &NeoAccount{w, eth}
	return
}

func GasToInt64(gas float64) int64 {
	return helper.Fixed8FromFloat64(gas).Value
}

func (acc *NeoAccount) GetWallet() *wallet.Account {
	return acc.neoWallet
}

// GenerateNewNeoWallet creates a new .json wallet file
func GenerateNewNeoWallet(path, password, wif string) (err error) {
	w := wallet.NewWallet()
	if wif != "" {
		err = w.ImportFromWIF(wif)
		if err != nil {return}
	}
	err = w.EncryptAll(password)
	if err != nil {return}
	err = w.Save(path)
	return
}

// Needed because file signing is done with Eth keys
type EthereumKey struct {
	PrivateKey []byte	// 32
	PublicKey  []byte	// 64
}

func ToEcdsa(privateKey []byte) (ethKey *EthereumKey, err error) {
	point, err := ecrypto.ToECDSA(privateKey)
	if err != nil {return}

	publicKeyStringX := hexutil.EncodeBig(point.PublicKey.X)
	publicKeyStringY := hexutil.EncodeBig(point.PublicKey.Y)
	for _, p := range []*string {&publicKeyStringX,&publicKeyStringY} {
		for len(*p) < 66 {
			*p = "0x" + "0" + strings.Replace(*p, "0x", "", -1)
		}
	}
	publicKey := hexutil.MustDecode(publicKeyStringX + strings.Replace(publicKeyStringY, "0x", "", -1))
	ethKey = new(EthereumKey)
	ethKey.PrivateKey = privateKey[0:32]
	ethKey.PublicKey = publicKey[0:64]
	return
}

func GasPerGByteFromProfile(gasPerMByte int64) string {
	return humanize.CommafWithDigits(float64(gasPerMByte)*Kilo, 4)
}
