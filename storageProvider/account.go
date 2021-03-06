package storageProvider

import (
	"math"
	"os"
	fp "path/filepath"
	"strconv"
	"strings"
	"time"

	dht "github.com/archoncloud/archon-dht/archon"
	dhtp "github.com/archoncloud/archon-dht/dht_permission_layers"
	"github.com/archoncloud/archoncloud-ethereum/rpc_utils"
	"github.com/archoncloud/archoncloud-go/account"
	"github.com/archoncloud/archoncloud-go/blockchainAPI/neo"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/pariz/gountries"
)

// For info only
var StartTime time.Time
var PortsUsed []int

const (
	StorageRoot    = "www"
	ShardsFolder   = "shards"
	HashesFolder   = "hashes"
	LogFolder      = "log"
	ActivityFolder = "activity"
)

// where the executable resides. Current directory is not used
var rootFolder string

func RelativeToRoot(absPath string) string {
	return strings.TrimPrefix(absPath, rootFolder)
}

func GetShardsFolder() string {
	return fp.Join(rootFolder, StorageRoot, ShardsFolder)
}

func GetHashesFolder() string {
	return fp.Join(rootFolder, StorageRoot, HashesFolder)
}

func GetActivityFolder() string {
	return fp.Join(rootFolder, StorageRoot, ActivityFolder)
}

func GetLogFilePath() string {
	return fp.Join(rootFolder, LogFolder, "sp.log")
}

func InRootFolder(paths ...string) string {
	combined := rootFolder
	for _, p := range paths {
		combined = fp.Join(combined, p)
	}
	return combined
}

func InShardsFolder(paths ...string) string {
	combined := GetShardsFolder()
	for _, p := range paths {
		combined = fp.Join(combined, p)
	}
	return combined
}

func InHashesFolder(paths ...string) string {
	combined := GetHashesFolder()
	for _, p := range paths {
		combined = fp.Join(combined, p)
	}
	return combined
}

func AccessedPath(acl UploadAccessControlLevel, spath string) string {
	var ret string
	switch acl {
	case Priv_RestrictedGroup:
		ret = fp.Join(SPriv_RestrictedGroup, spath)
		break
	case Public:
		ret = fp.Join(SPublic, spath)
		break
	default:
		ret = fp.Join(SPriv_UploaderOnly, spath)
		break
	}
	return ret
}

// StoredFilePath returns the path where the file is stored on the server
// If shardIndex is negative, this is a whole file
func StoredFilePath(a *ArchonUrl, acl UploadAccessControlLevel, shardIndex int) string {
	spath := a.ShardPath(shardIndex)
	accessedPath := AccessedPath(acl, spath)
	if a.IsHash() {
		// hash URL
		return InHashesFolder(accessedPath)
	}
	return InShardsFolder(accessedPath)
}

// Make sure root subfolders exist
func makeFolders() {
	folders := []string{GetShardsFolder(), GetHashesFolder(), GetActivityFolder()}
	err := MakeFolders(folders)
	Abort(err)
}

// The Storage Provider account
var SPAccount struct {
	Eth *account.EthAccount
	Neo *account.NeoAccount
}

func GetAccount(accT interfaces.AccountType) interfaces.IAccount {
	switch accT {
	case interfaces.EthAccountType:
		return SPAccount.Eth
	case interfaces.NeoAccountType:
		return SPAccount.Neo
	default:
		return nil
	}
}

func SetupAccountAndDht() {
	StartTime = time.Now()
	makeFolders()
	// CLI arguments
	ProcessArgs()
	conf := GetSPConfiguration()

	regInfo, err := account.GetRegistrationInfo("")
	Abort(err)
	country, _ := gountries.New().FindCountryByAlpha(regInfo.CountryA3)
	apiPorts := ApiPorts(conf)
	urls := Urls{Host: conf.Host, HttpPort: strconv.Itoa(apiPorts[0])}
	if len(apiPorts) > 1 {
		urls.HttpsPort = strconv.Itoa(apiPorts[1])
	}
	LogInfo.Printf("Storage Provider V%s starting\n", Version)

	var multiAddrString string
	if conf.Host == "localhost" {
		// This is for debugging only
		multiAddrString = "/ip4/" + Localhost
	} else {
		hostMa, err := GetMultiAddressOf(conf.Host)
		Abort(err)
		multiAddrString = hostMa.String()
	}
	multiAddrString += "/tcp/"
	LogInfo.Printf("Host multi address is: %s\n", multiAddrString)

	if conf.EthWalletPath != "" {
		password := GetPassword("Ethereum", showPassword)
		acc, err := account.NewEthAccount(DefaultToExecutable(conf.EthWalletPath), password)
		if err == nil {
			SPAccount.Eth = acc
		} else {
			LogError.Println(err.Error())
		}
	}
	if conf.NeoWalletPath != "" {
		password := GetPassword("Neo", showPassword)
		acc, err := account.NewNeoAccount(conf.NeoWalletPath, password)
		if err == nil {
			SPAccount.Neo = acc
		} else {
			LogError.Println(err.Error())
		}
	}

	dhtPort := apiPorts[0] + 3
	publicDht := dht.DHTConnectionConfig{
		RandomInt64(0, math.MaxInt64), // Seed
		true,                          // Global
		false,                         // IAmBootstrap
		true,                          // OptInToNetworking
		country.Codes,                 // SelfReportedCountryCode
		dhtp.NonPermissioned{},
		urls.String(), // Url
		multiAddrString + strconv.Itoa(dhtPort),
		conf.BootstrapPeers,
	}
	PortsUsed = append(PortsUsed, dhtPort)
	dhtPort++
	dhtConf := []dht.DHTConnectionConfig{publicDht}

	if SPAccount.Eth != nil {
		if len(config.EthRpcUrls) == 0 {
			AbortWithString("The eth_rpc_urls section in the config file cannot be empty")
		}
		err = rpc_utils.SetRpcUrl(config.EthRpcUrls)
		Abort(err)
		ethDht := dht.DHTConnectionConfig{
			interfaces.GetSeed(SPAccount.Eth),
			true,
			false,
			true,
			country.Codes,
			dhtp.Ethereum{},
			urls.String(),
			multiAddrString + strconv.Itoa(dhtPort),
			conf.EthBootstrapPeers,
		}
		dhtConf = append(dhtConf, ethDht)
		PortsUsed = append(PortsUsed, dhtPort)
		dhtPort++
		checkRegistration(SPAccount.Eth, regInfo)
		LogInfo.Printf("Eth account initialized. Address: %s", SPAccount.Eth.AddressString())
	}

	if SPAccount.Neo != nil {
		if len(config.NeoRpcUrls) == 0 {
			AbortWithString("The neo_rpc_urls section in the config file cannot be empty")
		}
		if neo.SetRpcUrl(config.NeoRpcUrls) == "" {
			AbortWithString("None of the neo_rpc_urls is responding")
		}
		neoDht := dht.DHTConnectionConfig{
			interfaces.GetSeed(SPAccount.Neo),
			true,
			false,
			true,
			country.Codes,
			dhtp.Neo{},
			urls.String(),
			multiAddrString + strconv.Itoa(dhtPort),
			conf.NeoBootstrapPeers,
		}
		dhtConf = append(dhtConf, neoDht)
		PortsUsed = append(PortsUsed, dhtPort)
		dhtPort++
		checkRegistration(SPAccount.Neo, regInfo)
		LogInfo.Printf("Neo account initialized. Address: %s", SPAccount.Neo.AddressString())
	}

	if len(dhtConf) == 1 {
		LogWarning.Println("No Ethereum or Neo wallets provided")
	}
	d, err := dht.Init(dhtConf, dhtPort)
	Abort(err)
	PortsUsed = append(PortsUsed, dhtPort)
	dhtInstance = d

	if isInfoCmd {
		time.Sleep(5 * time.Second) //TODO: should not need this
		showInfo()
		os.Exit(0)
	}
}

func checkRegistration(acc interfaces.IAccount, regInfo *interfaces.RegistrationInfo) {
	isRegistered := acc.IsSpRegistered()
	if isUnregisterCmd {
		unregisterSp(acc, isRegistered)
		// After unregister we do not need to continue
		os.Exit(0)
	}
	if isRegisterCmd {
		if isRegistered {
			// First unregister
			LogInfo.Printf("Unregistering before registering (%s)\n", acc.BlockchainName())
			unregisterSp(acc, true)
		}
		resisterSp(acc, regInfo)
	} else if !isRegistered {
		if inBatchMode || Yes("This account is not registered on "+acc.BlockchainName()+". Do you want to register") {
			resisterSp(acc, regInfo)
		} else {
			os.Exit(0)
		}
	}
}

func resisterSp(acc interfaces.IAccount, regInfo *interfaces.RegistrationInfo) {
	LogInfo.Printf("Registering on %s\n", acc.BlockchainName())
	txId, err := acc.RegisterSP(regInfo)
	Abort(err)
	LogInfo.Println("Registration Tx Id=", txId)
}

func unregisterSp(acc interfaces.IAccount, isRegistered bool) {
	if isRegistered {
		err := acc.UnregisterSP()
		if err != nil {
			AbortWithString("unregister failed: " + err.Error())
		}
	}
	LogInfo.Printf("Unregister succeeded (%s)\n", acc.BlockchainName())
}
