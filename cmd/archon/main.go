// The Archon upload/download client
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/archoncloud/archoncloud-ethereum/rpc_utils"
	"github.com/archoncloud/archoncloud-go/account"
	"github.com/archoncloud/archoncloud-go/blockchainAPI/neo"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/download"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/archoncloud/archoncloud-go/upload"
	"github.com/jessevdk/go-flags"
)

var isUpload, isDownload bool

type Configuration struct {
	WalletPath          string                   `json:"wallet_path"`
	PreferHttp          bool                     `json:"http"`
	Overwrite           bool                     `json:"overwrite"`
	Encoding            string                   `json:"encoding"`
	HashUrl             bool                     `json:"hashUrl"`
	ReedSolomonRequired int                      `json:"rs_required"`
	ReedSolomonTotal    int                      `json:"rs_total"`
	AccessControlLevel  UploadAccessControlLevel `json:"access_control_level"`
	DownloadDir         string                   `json:"download_dir"`
	// The following can only be edited manually
	EthRpcUrls []string `json:"eth_rpc_urls"`
	NeoRpcUrls []string `json:"neo_rpc_urls"`
}

func (c *Configuration) String() string {
	s := fmt.Sprintf("wallet_path=%q, http=%v, overwrite=%v, hashUrl=%v, encoding=%s",
		c.WalletPath, c.PreferHttp, c.Overwrite, c.HashUrl, c.Encoding)
	if strings.HasPrefix(c.Encoding, "RS") {
		s += fmt.Sprintf(", rs_required=%d, rs_total=%d", c.ReedSolomonRequired, c.ReedSolomonTotal)
	}
	if c.DownloadDir != "" {
		s += fmt.Sprintf(", download_dir=%q", c.DownloadDir)
	}
	return s
}

func newConfiguration() *Configuration {
	// default
	conf := Configuration{
		"",
		true,
		true,
		"mxor",
		false,
		3,
		8,
		Priv_UploaderOnly,
		"",
		nil,
		neo.RpcUrls(),
	}
	err := GetAppConfiguration(&conf)
	if !errors.Is(err, os.ErrNotExist) {
		Abort(err)
	}
	return &conf
}

type UploadCommand struct {
}

func (x *UploadCommand) Execute(args []string) error {
	isUpload = true
	return nil
}

type DownloadCommand struct {
}

func (x *DownloadCommand) Execute(args []string) error {
	isDownload = true
	return nil
}

type genEthWalletCommand struct{}

func (x *genEthWalletCommand) Execute(args []string) error {
	// Generate the wallet and return
	account.GenerateNewWalletFile(interfaces.EthAccountType, false)
	os.Exit(0)
	return nil
}

type genNeoWalletCommand struct{}

func (x *genNeoWalletCommand) Execute(args []string) error {
	// Generate the wallet and return
	account.GenerateNewWalletFile(interfaces.NeoAccountType, false)
	os.Exit(0)
	return nil
}

type versionCommand struct{}

func (x *versionCommand) Execute(args []string) error {
	fmt.Printf("V%s\n", Version)
	os.Exit(0)
	return nil
}

func main() {
	var options struct {
		PreferHttp         *bool                     `long:"http" description:"Prefer connecting over HTTP"`
		Overwrite          *bool                     `short:"o" long:"overwrite" description:"Overwrite existing file"` // default false
		AccessControlLevel *UploadAccessControlLevel `short:"a" long:"accesscontrollevel" description:"file access levels: 0->UploaderOnly, 1->PrivateRestrictedGroup, 2->Public. Default is 0"`
		Wallet             string                    `long:"wallet" description:"Path to Ethereum or Neo wallet file"`
		PasswordFile       *string                   `long:"passwordFile" description:"Path to the password file for wallet.\nCan be relative if in executable folder\nIf set, will run in batch mode"`
		UploadGroup        struct {
			File     string  `short:"f" long:"file" description:"Path of file to upload"`
			Encoding *string `short:"e" long:"encoding" choice:"none" choice:"mxor" choice:"RSa" choice:"RSb" description:"How the file is stored"`
			RSReq    *int    `long:"req" description:"Number of shards required for reconstruction (for RS... only)"`
			RSTot    *int    `long:"tot" description:"Total number of shards. Must be larger than req"`
			Type     *string `short:"t" long:"type" choice:"hash" choice:"named" description:"The kind of Archon Url returned"`
			CloudDir *string `long:"cloudDir" description:"the path of the folder in the cloud"`
		} `group:"Options for upload"`
		DownloadGroup struct {
			Url              string  `short:"u" long:"url" description:"Archon URL of the file to download"`
			DownloadDir      *string `short:"d" long:"downloadDir" description:"The folder for download"`
			DownloadFileName string  `long:"downloadName" description:"The file name, if different from the URL name"`
		} `group:"Options for download"`
	}

	parser := flags.NewParser(&options, flags.Default)
	_, _ = parser.AddCommand("upload",
		"Upload a file and get an Archon URL",
		"",
		&UploadCommand{})
	_, _ = parser.AddCommand("download",
		"Download a file from an Archon URL",
		"",
		&DownloadCommand{})
	_, _ = parser.AddCommand("version",
		"Print version and exit",
		"",
		&versionCommand{})
	_, _ = parser.AddCommand("generateEthWalletFile",
		"Generates a new Ethereum .json wallet file, with a new address",
		"",
		&genEthWalletCommand{})
	_, _ = parser.AddCommand("generateNeoWalletFile",
		"Generates a new Neo .json wallet file",
		"",
		&genNeoWalletCommand{})

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			Abort(err)
		}
	}

	if !isDownload && !isUpload {
		InvalidArgs("You need to specify a command")
	}

	conf := newConfiguration()
	confChanged := false
	if options.Wallet != "" {
		conf.WalletPath = options.Wallet
		confChanged = true
	}
	if options.PreferHttp != nil {
		conf.PreferHttp = *options.PreferHttp
		confChanged = true
	}
	if options.Overwrite != nil && conf.Overwrite != *options.Overwrite {
		conf.Overwrite = *options.Overwrite
		confChanged = true
	}
	if options.AccessControlLevel != nil && conf.AccessControlLevel != *options.AccessControlLevel {
		conf.AccessControlLevel = *options.AccessControlLevel
	}
	if options.UploadGroup.Type != nil {
		isHash := *options.UploadGroup.Type == "hash"
		if conf.HashUrl != isHash {
			conf.HashUrl = isHash
			confChanged = true
		}
	}
	if options.UploadGroup.Encoding != nil && *options.UploadGroup.Encoding != conf.Encoding {
		conf.Encoding = *options.UploadGroup.Encoding
		confChanged = true
	}
	if options.UploadGroup.RSReq != nil && *options.UploadGroup.RSReq != conf.ReedSolomonRequired {
		conf.ReedSolomonRequired = *options.UploadGroup.RSReq
		confChanged = true
	}
	if options.UploadGroup.RSTot != nil && *options.UploadGroup.RSTot != conf.ReedSolomonTotal {
		conf.ReedSolomonTotal = *options.UploadGroup.RSTot
		confChanged = true
	}

	if options.DownloadGroup.DownloadDir != nil && *options.DownloadGroup.DownloadDir != conf.DownloadDir {
		conf.DownloadDir = *options.DownloadGroup.DownloadDir
		confChanged = true
	}

	batch := options.PasswordFile != nil

	fmt.Printf("Configuration is:\n    %s\n", conf.String())
	if confChanged {
		err := SaveAppConfiguration(conf)
		Abort(err)
	}

	if isUpload {
		// defaults
		if conf.Encoding == "" {
			conf.Encoding = "none"
		}

		if options.UploadGroup.File == "" {
			InvalidArgs("You need to specify a file")
		}
		acc := conf.getAccount(options.PasswordFile)
		req := upload.Request{
			options.UploadGroup.File,
			"",
			conf.Encoding,
			conf.ReedSolomonTotal,
			conf.ReedSolomonRequired,
			conf.Overwrite,
			conf.AccessControlLevel,
			conf.HashUrl,
			conf.PreferHttp,
			acc,
			batch,
			0,
		}
		if options.UploadGroup.CloudDir != nil {
			req.CloudDir = strings.ReplaceAll(*options.UploadGroup.CloudDir, "\\", "/")
		}
		downloadUrl, _, err := req.Upload()
		Abort(err)
		fmt.Printf("Upload completed\nDownload URL is=%s\n", downloadUrl)
	} else {
		// Download
		u := options.DownloadGroup.Url
		if u == "" {
			InvalidArgs("You need to specify the Archon URL")
		}
		aUrl, err := NewArchonUrl(u)
		Abort(err)

		if !aUrl.IsHash() {
			account := conf.getAccount(options.PasswordFile)
			userName, err := account.GetUserName()
			Abort(err)
			if aUrl.Username != userName {
				Abort(fmt.Errorf("account username (%s) does not match the URL username (%s)", userName, aUrl.Username))
			}
		}

		req := download.Request{
			aUrl,
			conf.DownloadDir,
			options.DownloadGroup.DownloadFileName,
			conf.Overwrite,
			conf.PreferHttp,
			batch,
		}
		err = req.Download()
		Abort(err)
	}
}

func (conf *Configuration) getAccount(passwordFile *string) interfaces.IAccount {
	if conf.WalletPath == "" {
		InvalidArgs("You need to specify the wallet path")
	}
	var password string
	if passwordFile != nil {
		p, err := ioutil.ReadFile(DefaultToExecutable(*passwordFile))
		Abort(err)
		password = strings.TrimSpace(string(p))
	} else {
		password = GetPassword("Wallet", false)
	}

	acc, err := account.NewIAccount(conf.WalletPath, password)
	Abort(err)
	switch acc.GetAccountType() {
	case interfaces.EthAccountType:
		if conf.EthRpcUrls == nil {
			AbortWithString("eth_rpc_urls needs to be filled in")
		}
		rpc_utils.SetRpcUrl(conf.EthRpcUrls)
	case interfaces.NeoAccountType:
		if neo.SetRpcUrl(conf.NeoRpcUrls) == "" {
			AbortWithString("None of the neo_rpc_urls is responding")
		}
	default:
		AbortWithString("unknown account type")
	}
	userName, err := acc.GetUserName()
	if userName == "" {
		// Need to register
		userName = PromptForInput("You need to register a user name.\nUser name:")
		err := acc.RegisterUserName(userName)
		Abort(err)
	}
	fmt.Printf("User name is %q\n", userName)
	return acc
}
