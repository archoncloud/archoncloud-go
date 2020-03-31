package storageProvider

import (
	"github.com/itsmeknt/archoncloud-go/account"
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/itsmeknt/archoncloud-go/interfaces"
	dht "github.com/itsmeknt/archoncloud-go/networking/archon-dht"
	dhtp "github.com/itsmeknt/archoncloud-go/networking/archon-dht/permission_layers"
	"github.com/pariz/gountries"
	"math"
	"os"
	fp "path/filepath"
	"strconv"
	"strings"
	"time"
)

const SPVersion = "0.25"

// For info only
var StartTime time.Time
var PortsUsed []int

const (
	StorageRoot     = "www"
	ShardsFolder    = "shards"
	HashesFolder    = "hashes"
	LogFolder		= "log"
	ActivityFolder	= "activity"
)

// where the executable resides. Current directory is not used
var rootFolder string;

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

// StoredFilePath returns the path where the file is stored on the server
// If shardIndex is negative, this is a whole file
func StoredFilePath(a *ArchonUrl, shardIndex int) string {
	spath := a.ShardPath(shardIndex)
	if a.IsHash() {
		// hash URL
		return InHashesFolder(spath)
	}
	return InShardsFolder(spath)
}

// Make sure root subfolders exist
func makeFolders() {
	folders := []string{GetShardsFolder(),GetHashesFolder(),GetActivityFolder()}
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
	case interfaces.EthAccountType: return SPAccount.Eth
	case interfaces.NeoAccountType: return SPAccount.Neo
	default: return nil
	}
}

func SetupAccountAndDht() {
	StartTime = time.Now()
	makeFolders()
	// CLI arguments
	ProcessArgs()
	conf := GetSPConfiguration()

	regInfo, err := account.GetRegistrationInfo()
	Abort(err)
	country, _ := gountries.New().FindCountryByAlpha(regInfo.CountryA3)
	apiPorts := ApiPorts(conf)
	urls := Urls{Host:conf.Host, HttpPort:strconv.Itoa(apiPorts[0])}
	if len(apiPorts) > 1 {
		urls.HttpsPort = strconv.Itoa(apiPorts[1])
	}
	LogInfo.Printf("Storage Provider V%s starting\n", SPVersion)

	var multiAddrString string
	if conf.Host == "localhost" {
		// This is for debugging only
		multiAddrString = "/ip4/192.168.1.161"
	} else {
		hostMa, err := GetMultiAddressOf(conf.Host)
		Abort(err)
		multiAddrString = hostMa.String()
	}
	multiAddrString += "/tcp/"
	LogInfo.Printf("Host multi address is: %s\n", multiAddrString)

	if conf.EthWalletPath != "" {
		var password string
		if debug {
			// To avoid having to type this
			password = "ethTestingWallet"
		} else {
			password = GetPassword("Ethereum", showPassword)
		}
		acc, err := account.NewEthAccount(DefaultToExecutable(conf.EthWalletPath), password)
		Abort(err)
		SPAccount.Eth = acc
	}
	if conf.NeoWalletPath != "" {
		var password string
		if debug {
			// To avoid having to type this
			password = "archon"
		} else {
			password = GetPassword("Neo", showPassword)
		}
		acc, err := account.NewNeoAccount(conf.NeoWalletPath, password)
		Abort(err)
		SPAccount.Neo = acc
	}

	dhtPort := apiPorts[0]+3
	publicDht := dht.DHTConnectionConfig{
		RandomInt64(0, math.MaxInt64),	// Seed
		true,                          		// Global
		false,                      	// IAmBootstrap
		nil,
		true,               	// OptInToNetworking
		country.Codes,             	// SelfReportedCountryCode
		dhtp.NonPermissioned{},
		urls.String(),               			// Url
		multiAddrString + strconv.Itoa(dhtPort),
		conf.BootstrapPeers,
	}
	PortsUsed = append(PortsUsed,dhtPort)
	dhtPort++
	dhtConf := []dht.DHTConnectionConfig{publicDht}

	if SPAccount.Eth != nil {
		ethDht := dht.DHTConnectionConfig{
			interfaces.GetSeed(SPAccount.Eth),
			true,
			false,
			SPAccount.Eth,
			true,
			country.Codes,
			dhtp.Ethereum{},
			urls.String(),
			multiAddrString + strconv.Itoa(dhtPort),
			conf.EthBootstrapPeers,
		}
		dhtConf = append(dhtConf, ethDht)
		PortsUsed = append(PortsUsed,dhtPort)
		dhtPort++
		checkRegistration(SPAccount.Eth, regInfo)
		LogInfo.Printf("Eth account initialized. Address: %s", SPAccount.Eth.AddressString())
	}

	if SPAccount.Neo != nil {
		neoDht := dht.DHTConnectionConfig{
			interfaces.GetSeed(SPAccount.Neo),
			true,
			false,
			SPAccount.Neo,
			true,
			country.Codes,
			dhtp.Neo{},
			urls.String(),
			multiAddrString + strconv.Itoa(dhtPort),
			conf.NeoBootstrapPeers,
		}
		dhtConf = append(dhtConf, neoDht)
		PortsUsed = append(PortsUsed,dhtPort)
		dhtPort++
		checkRegistration(SPAccount.Neo, regInfo)
		LogInfo.Printf("Neo account initialized. Address: %s", SPAccount.Neo.AddressString())
	}

	if len(dhtConf) == 1 {
		LogWarning.Println("No Ethereum or Neo wallets provided")
	}
	d, err := dht.Init(dhtConf, dhtPort)
	Abort(err)
	PortsUsed = append(PortsUsed,dhtPort)
	dhtInstance = d

	if isInfoCmd {
		time.Sleep(5*time.Second)	//TODO: should not need this
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
			unregisterSp(acc,true)
		}
		resisterSp(acc,regInfo)
	} else if !isRegistered {
		if inBatchMode || Yes("This account is not registered on " + acc.BlockchainName() + ". Do you want to register") {
			resisterSp(acc,regInfo)
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
