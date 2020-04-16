package account

import (
	"fmt"
	dht "github.com/archoncloud/archon-dht/archon"
	"github.com/dustin/go-humanize"
	"github.com/archoncloud/archoncloud-go/blockchainAPI/neo"
	. "github.com/archoncloud/archoncloud-go/common"
	ifc "github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/archoncloud/archon-dht/permission_layer"
	"github.com/archoncloud/archoncloud-go/shards"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func NewIAccount(walletPath string, password string) (iaccount ifc.IAccount, err error) {
	path := DefaultToExecutable(walletPath)
	if !FileExists(path) {
		err = fmt.Errorf("Cannot find file %q", path)
		return
	}
	buf, err := ioutil.ReadFile(DefaultToExecutable(walletPath))
	if err != nil {return}
	s := string(buf)
	if strings.Contains(s, `"contract"`) && strings.Contains(s, `"scrypt"`) {
		// Neo
		return NewNeoAccount(walletPath,password)
	}
	if strings.Contains(s, `"crypto"`) && strings.Contains(s, `"ciphertext"`) {
		// Eth
		return NewEthAccount(walletPath,password)
	}
	err = fmt.Errorf("unknown wallet type")
	return
}

func ArchonSignatureFor(r io.Reader, acc ifc.IAccount) (string, []byte, int64,  error) {
	hash, n, err := GetArchonHashOf(r)
	if err != nil {return "", nil, 0, err}

	sig, err := acc.Sign(hash)
	return BytesToString(hash), sig, n, err
}

func PermLayerID(acc ifc.IAccount) permission_layer.PermissionLayerID {
	if acc != nil {
		switch acc.GetAccountType() {
		case ifc.EthAccountType: return permission_layer.EthPermissionId
		case ifc.NeoAccountType: return permission_layer.NeoPermissionId
		}
	}
	return permission_layer.NotPermissionId
}

func IsEth(acc ifc.IAccount) bool {
	return acc.GetAccountType() == ifc.EthAccountType;
}

func PrettyCurrencyForAccount(acc ifc.IAccount, amount int64) string {
	var p string
	switch acc.GetAccountType() {
	case ifc.EthAccountType:
		p = WeiString(amount)
	case ifc.NeoAccountType:
		p = neo.CgasString(amount)
	default:
		return ""
	}
	f := float64(amount)
	f /= float64(acc.HundredthOfCent() * 100)
	if f > 100.0 {
		dollars := humanize.CommafWithDigits(f/100.0, 2)
		p += fmt.Sprintf(" ($%s)", dollars)
	} else {
		cents := humanize.CommafWithDigits(f, 6)
		p += fmt.Sprintf(" (%sc)", cents)
	}
	return p
}

func confirmPrice(acc ifc.IAccount, numBytes int64, sps StorageProviders, maxPayment int64) (price int64, err error) {
	// For Eth, the price is in Wei for Neo in CGAS
	price = sps.PriceOfUpload(numBytes)
	if maxPayment > 0 {
		if maxPayment < price {
			err = fmt.Errorf("price (%d) exceeds MaxPayment (%d)", price, maxPayment)
			return
		}
	}
	priceS := acc.PrettyCurrency(price)
	if maxPayment >= 0 {
		fmt.Printf("Charge is=%s\n", priceS)
	} else if !Yes(fmt.Sprintf("You will be charged %s for the upload. Is this OK?", priceS)) {
		// Rejected
		os.Exit(2)
	}
	return
}

func GenerateNewWalletFile(accT ifc.AccountType, showPassword bool) {
	var ns, nl, cur string
	switch accT {
	case ifc.EthAccountType:
		ns = "Eth"
		nl = "Ethereum"
		cur = "Ethers"
	case ifc.NeoAccountType:
		ns = "Neo"
		nl = "Neo"
		cur = "Gas"
	default: return
	}
	path := DefaultToExecutable(fmt.Sprintf("new%sWallet.json", ns))
	password := GetPassword("Password for new Ethereum wallet to be generated", showPassword)
	if password == "" {
		AbortWithString("Aborted")
	}
	confirm := GetPassword("Confirm passord", showPassword)
	if password != confirm {
		AbortWithString("Passwords don't match")
	}
	var err error
	private := ""
	switch accT {
	case ifc.EthAccountType:
		if Yes("Do you want to provide an existing private key") {
			private = PromptForInput("Enter your private key")
			if private == "" {
				AbortWithString("Aborted")
			}
		}
		err = GenerateNewEthWallet(path,password,private)
	case ifc.NeoAccountType:
		// The WIF format is to add prefix 0x80 and suffix 0x01 in the original 32 bytes private key, and get string of Base58Check encoding.
		// private key: 32 bytes (UInt256)
		if Yes("Do you want to provide an existing WIF string") {
			private = PromptForInput("Enter your WIF string")
			if private == "" {
				AbortWithString("Aborted")
			}
		}
		err = GenerateNewNeoWallet(path,password,private)
	}
	Abort(err)
	acc, err := NewIAccount(path, password)
	Abort(err)
	fmt.Printf("%s wallet generated in %q\n", nl, path)
	fmt.Printf("Please remember the password you just entered\n")
	fmt.Printf("Rename this file as you wish and then enter it in the .config file\n")
	if private != "" {
		fmt.Printf("You may need to transfer %s into it before you can use it\n", cur)
		fmt.Printf("The account address is: %q\n", acc.AddressString())
	}
}

// if maxPayment is negative, then the user will be prompted
func ProposeUpload(acc ifc.IAccount, fc *shards.FileContainer, s *shards.ShardsContainer, a *ArchonUrl, sps StorageProviders, maxPayment int64) (txId string, price int64, err error) {
	switch acc.GetAccountType() {
	case ifc.EthAccountType:
		ea := acc.(*EthAccount)
		return ea.ProposeUpload(fc,s, a, sps, maxPayment)
	case ifc.NeoAccountType:
		na := acc.(*NeoAccount)
		txIds, price2, err2 := na.ProposeUpload(fc,s, a, sps, maxPayment)
		if err2 != nil {
			err = err2
			return
		}
		price = price2
		if len(sps) == 1 {
			txId = txIds[sps[0].Address]
		}
	}
	err = errors.New("unknown account type")
	return
}

func GetNodeId(acc ifc.IAccount) (nodeId string, err error) {
	nId, err := dht.GetNodeID(ifc.GetSeed(acc))
	if err != nil {return}
	nodeId = nId.Pretty()
	return
}