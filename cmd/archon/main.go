// The Archon upload/download client
package main

import (
	"errors"
	"fmt"
	"github.com/archoncloud/archoncloud-go/account"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/download"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/archoncloud/archoncloud-go/upload"
	"github.com/jessevdk/go-flags"
	"io/ioutil"
	"os"
	"strings"
)

/*
for Uploader client:
1. on initialization:
 -open ethereum wallet
 -check if ethereum wallet is already registered with a username
 -if so, display username, if not, display "not yet registered"
 -download list of SPs registered on the smart contract, then display their upload URLs
2. registerUsername function: given a username string, sends a registerUsername tx.
 -filter for the username. illegal characters: / . # @ ~ * < > : " ?
3. upload function: given file data, archon filepath, encodingOptions (right now, just encoded/notencoded but more options in the future), and a list of storage provider ethereum addresses to upload to
 -filter archon filepath. illegal characters: # ~ * < > : " ?
 -the way the upload works is:
for (int i = 0; i < shards.length; i++) {
 byte* shard = shards[i];
 SP sp = listOfSp[i % listOfSp.length];
 upload(shard, sp);
}
 -if the file is unencoded, treat that as just a 1 shard file
 -the upload function will return an archon named URL and archon hash URL on completion, generated client-side and possibly double checked from the response of the SP (edited)
*/

var isUpload, isDownload bool

type Configuration struct {
	WalletPath          string `json:"wallet_path"`
	PreferHttp          bool   `json:"http"`
	Overwrite           bool   `json:"overwrite"`
	Encoding            string `json:"encoding"`
	HashUrl             bool   `json:"hashUrl"`
	ReedSolomonRequired int    `json:"rs_required"`
	ReedSolomonTotal    int    `json:"rs_total"`
	DownloadDir			string `json:"download_dir"`
}

func (c *Configuration) String() string {
	s := fmt.Sprintf("wallet_path=%q, http=%v, overwrite=%v, hashUrl=%v, encoding=%s",
		c.WalletPath, c.PreferHttp, c.Overwrite, c.HashUrl, c.Encoding)
	if strings.HasPrefix(c.Encoding,"RS") {
		s += fmt.Sprintf(", rs_required=%d, rs_total=%d", c.ReedSolomonRequired, c.ReedSolomonTotal)
	}
	if c.DownloadDir != "" {
		s += fmt.Sprintf(", download_dir=%q", c.DownloadDir)
	}
	return s;
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
		"",
	}
	err := GetAppConfiguration(&conf)
	if !errors.Is(err, os.ErrNotExist ) {
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

var isGenEthWalletCmd bool
type genEthWalletCommand struct {
}

func (x *genEthWalletCommand) Execute(args []string) error {
	isGenEthWalletCmd = true
	return nil
}

func main() {
	var options struct {
		PreferHttp *bool `long:"http" description:"Prefer connecting over HTTP"`
		Overwrite *bool `short:"o" long:"overwrite" description:"Overwrite existing file"`	// default false
		Wallet string `long:"wallet" description:"Path to Ethereum or Neo wallet file"`
		PasswordFile *string  `long:"passwordFile" description:"Path to the password file for wallet.\nCan be relative if in executable folder\nIf set, will run in batch mode"`
		UploadGroup struct {
			File string `short:"f" long:"file" description:"Path of file to upload"`
			Encoding *string `short:"e" long:"encoding" choice:"none" choice:"mxor" choice:"RSa" choice:"RSb" description:"How the file is stored"`
			RSReq *int `long:"req" description:"Number of shards required for reconstruction (for RS... only)"`
			RSTot *int `long:"tot" description:"Total number of shards. Must be larger than req"`
			Type *string `short:"t" long:"type" choice:"hash" choice:"named" description:"The kind of Archon Url returned"`
			CloudDir *string `long:"cloudDir" description:"the path of the folder in the cloud"`
		} `group:"Options for upload"`
		DownloadGroup struct {
			Url         string `short:"u" long:"url" description:"Archon URL of the file to download"`
			DownloadDir *string `short:"d" long:"downloadDir" description:"The folder for download"`
			DownloadFileName string `long:"downloadName" description:"The file name, if different from the URL name"`
		} `group:"Options for download"`
	}

	var uploadCommand UploadCommand
	var downloadCommand DownloadCommand
	var genEthWalletCmd genEthWalletCommand

	parser := flags.NewParser(&options, flags.Default)
	_, _ = parser.AddCommand("upload",
		"Upload a file and get an Archon URL",
		"",
		&uploadCommand)
	_, _ = parser.AddCommand("download",
		"Download a file from an Archon URL",
		"",
		&downloadCommand)
	_, _ = parser.AddCommand("generateEthWalletFile",
		"Generates a new Ethereum .json wallet file, with a new address",
		"",
		&genEthWalletCmd)

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			Abort(err)
		}
	}

	if isGenEthWalletCmd {
		// Generate the wallet and return
		account.GenerateNewWalletFile( interfaces.EthAccountType, false)
		os.Exit(0)
	}

	if !isDownload && !isUpload {
		InvalidArgs( "You need to specify a command")
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
		if conf.Encoding == "" {conf.Encoding = "none"}

		if options.UploadGroup.File == "" {
			InvalidArgs("You need to specify a file")
		}
		account := conf.getAccount(options.PasswordFile)

		req := upload.Request{
			options.UploadGroup.File,
			"",
			conf.Encoding,
			conf.ReedSolomonTotal,
			conf.ReedSolomonRequired,
			conf.Overwrite,
			conf.HashUrl,
			conf.PreferHttp,
			account,
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

	account, err := account.NewIAccount(conf.WalletPath, password)
	Abort(err)
	userName, err := account.GetUserName()
	Abort(err)
	if userName == "" {
		// Need to register
		userName = PromptForInput("You need to register a user name.\nUser name:")
		err := account.RegisterUserName(userName)
		Abort(err)
	}
	fmt.Printf("User name is %q\n", userName)
	return account
}
