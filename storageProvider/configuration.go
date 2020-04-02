package storageProvider

import (
	"errors"
	"fmt"
	"github.com/archoncloud/archoncloud-go/account"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type Configuration struct {
	HttpPort         string  `json:"http_port,omitempty"`	// obsolete
	EthWalletPath    string  `json:"eth_wallet_path"`
	NeoWalletPath    string  `json:"neo_wallet_path"`
	Host             string  `json:"host"` // DNS or IP
	Port			 string  `json:"port"` // the first of a consecutive range of 6 or more
	LogLevel         string  `json:"log_level"`
	// The following can only be edited manually
	BootstrapPeers	 []string `json:"bootstrap_peers"`
	EthBootstrapPeers	 []string `json:"eth_bootstrap_peers"`
	NeoBootstrapPeers	 []string `json:"neo_bootstrap_peers"`
}

var debug = false
var cfgOnce sync.Once
var config *Configuration

func bootStrapPeers() []string {
	switch BuildConfig {
	case Debug:  return []string{"/ip4/18.220.115.81/tcp/8001/ipfs/"+BootStrapNodeId}
	case Dev:  return []string{"/ip4/18.220.115.81/tcp/8001/ipfs/"+BootStrapNodeId}
	case Beta: return []string{"/ip4/18.220.115.81/tcp/9001/ipfs/"+BootStrapNodeId}
	default: return nil
	}
}

func ethBootStrapPeers() []string {
	switch BuildConfig {
	case Debug:  return []string{"/ip4/18.220.115.81/tcp/8002/ipfs/"+BootStrapNodeId}
	case Dev:  return []string{"/ip4/18.220.115.81/tcp/8002/ipfs/"+BootStrapNodeId}
	case Beta: return []string{"/ip4/18.220.115.81/tcp/9002/ipfs/"+BootStrapNodeId}
	default: return nil
	}
}

func neoBootStrapPeers() []string {
	switch BuildConfig {
	case Debug:  return []string{"/ip4/18.220.115.81/tcp/8003/ipfs/"+BootStrapNodeId}
	case Dev:  return []string{"/ip4/18.220.115.81/tcp/8003/ipfs/"+BootStrapNodeId}
	case Beta: return []string{"/ip4/18.220.115.81/tcp/9003/ipfs/"+BootStrapNodeId}
	default: return nil
	}
}

var inBatchMode = false
var showPassword = false

// GetSPConfiguration returns the current configuration
func GetSPConfiguration() *Configuration {
	cfgOnce.Do(func() {
		defaultConfig := Configuration{
			"",
			"",
			"",
			"",
			strconv.Itoa(SeedsPort()),
			"Info",
			bootStrapPeers(),
			ethBootStrapPeers(),
			neoBootStrapPeers(),
		}

		err := GetAppConfiguration(&config)
		if errors.Is(err, os.ErrNotExist ) {
			config = &defaultConfig
			// Save it
			err = SaveAppConfiguration(config)
		} else {
			Abort(err)
		}
	})

	return config
}

func (c *Configuration) String() string {
	s := fmt.Sprintf("host=%q port=%q loglevel=%q", c.Host, c.Port, c.LogLevel)
	if c.EthWalletPath != "" {
		s += fmt.Sprintf(" ethwallet=%q",c.EthWalletPath)
	}
	if c.NeoWalletPath != "" {
		s += fmt.Sprintf(" neowallet=%q",c.NeoWalletPath)
	}
	return s
}

var isRegisterCmd bool
type RegisterCommand struct {
}

func (x *RegisterCommand) Execute(args []string) error {
	isRegisterCmd = true
	return nil
}

var isUnregisterCmd bool
type UnregisterCommand struct {
}

func (x *UnregisterCommand) Execute(args []string) error {
	isUnregisterCmd = true
	return nil
}

var isInfoCmd bool
type infoCommand struct {
}

func (x *infoCommand) Execute(args []string) error {
	isInfoCmd = true
	return nil
}

var isGenEthWalletCmd bool
type genEthWalletCommand struct {
}

func (x *genEthWalletCommand) Execute(args []string) error {
	isGenEthWalletCmd = true
	return nil
}

var isGenNeoWalletCmd bool
type genNeoWalletCommand struct {
}

func (x *genNeoWalletCommand) Execute(args []string) error {
	isGenNeoWalletCmd = true
	return nil
}

func ProcessArgs() {
	ex, _ := os.Executable()
	rootFolder = filepath.Dir(ex)

	var options struct {
		Host		 string  `long:"host" description:"Host DNS or IP address"`
		Port 	 	string `long:"port" description:"First port of a free consecutive range of 7"`
		EthWallet    *string  `long:"ethwallet" description:"Path to Ethereum wallet file"`
		NeoWallet    *string  `long:"neowallet" description:"Path to Neo wallet file"`
		LogLevel     string  `short:"l" long:"loglevel" choice:"trace" choice:"debug" choice:"info" choice:"warning" choice:"error" description:"Level of logging"`
		ShowPassword *bool   `long:"show" description:"Show password while typing"`
		BatchMode	 bool	`long:"batch" description:"Batch mode. Provides default yes/no answers. To be used for deployment"`
	}

	parser := flags.NewParser(&options, flags.Default)
	parser.SubcommandsOptional = true

	var registerCmd RegisterCommand
	var unregisterCmd UnregisterCommand
	var infoCmd infoCommand
	var genEthWalletCmd genEthWalletCommand
	var genNeoWalletCmd genNeoWalletCommand
	_, _ = parser.AddCommand("register",
		"Register this storage provider. Reads data from registration.txt",
		"",
		&registerCmd)
	_, _ = parser.AddCommand("unregister",
		"Unregister this storage provider and exit",
		"",
		&unregisterCmd)
	_, _ = parser.AddCommand("info",
		"Print network min ask info and exit",
		"",
		&infoCmd)
	_, _ = parser.AddCommand("generateEthWalletFile",
		"Generates a new Ethereum .json wallet file",
		"",
		&genEthWalletCmd)
	_, _ = parser.AddCommand("generateNeoWalletFile",
		"Generates a new Neo .json wallet file",
		"",
		&genNeoWalletCmd)

	if remaining, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			Abort(err)
		}
	} else {
		// debug is a hidden command (an option really)
		if len(remaining) > 0 {
			if remaining[0] == "debug" {
				debug = true
				if len(remaining) > 1 {
					InvalidArgs("unknown argument: " + remaining[1])
				}
			} else {
				InvalidArgs("unknown argument: " + remaining[0])
			}
		}
	}

	if isGenEthWalletCmd {
		// Generate the wallet and return
		account.GenerateNewWalletFile(interfaces.EthAccountType, showPassword)
		os.Exit(0)
	}
	if isGenNeoWalletCmd {
		// Generate the wallet and return
		account.GenerateNewWalletFile(interfaces.NeoAccountType, showPassword)
		os.Exit(0)
	}

	conf := GetSPConfiguration()
	confChanged := false
	if options.EthWallet != nil && conf.EthWalletPath != *options.EthWallet {
		conf.EthWalletPath = *options.EthWallet
		confChanged = true
	}
	if options.NeoWallet != nil && conf.NeoWalletPath != *options.NeoWallet {
		conf.NeoWalletPath = *options.NeoWallet
		confChanged = true
	}
	if options.Host != "" {
		conf.Host = options.Host
		confChanged = true
	}
	if options.Port != "" {
		conf.Port = options.Port
		confChanged = true
	}
	if options.LogLevel != "" {
		conf.LogLevel = options.LogLevel
		confChanged = true
	}
	SetLoggingLevelFromName(conf.LogLevel)
	showPassword = options.ShowPassword != nil && *options.ShowPassword
	inBatchMode = options.BatchMode
	fmt.Printf("Configuration is:\n   %s\n", conf.String())
	if conf.Host == "" {
		Abort( errors.New("you need to specify the host name or IP"))
	}
	if conf.Port == "" {
		if conf.HttpPort == "" {
			Abort(errors.New("you need to specify a port"))
		}
		conf.Port = conf.HttpPort
		conf.HttpPort = "" // to remove, as it is obsolete
		confChanged = true
	}
	if len(conf.NeoBootstrapPeers) == 0 {
		// This was added later, so may be empty
		conf.NeoBootstrapPeers = neoBootStrapPeers()
		confChanged = true
	}
	if confChanged {
		err := SaveAppConfiguration(conf)
		Abort(err)
	}
	return
}