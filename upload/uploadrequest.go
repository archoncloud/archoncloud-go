// upload implements the Archon upload protocol
package upload

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	permLayer "github.com/archoncloud/archon-dht/permission_layer"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/archoncloud/archoncloud-go/shards"
	"github.com/pkg/errors"
)

const (
	EncodingNone = "none"
	EncodingMxor = "mxor"
	EncodingRSa  = "RSa" // archive-optimized
	EncodingRSb  = "RSb" // browser-optimized
)

type Request struct {
	FilePath        string
	CloudDir        string
	Encoding        string
	NumTotal        int // needed only when EncodingRSx
	NumRequired     int // needed only when EncodingRSx
	Overwrite       bool
	HashUrl         bool // user wants a hash URL, not a named URL
	PreferHttp      bool // user prefers using http, if available
	UploaderAccount interfaces.IAccount
	Batch           bool // operate in batch mode - no user interaction
	// If 0, any payment is valid, otherwise in units of blockchain
	// Eth: Wei, Neo: Gas* (10**8)
	MaxPayment  int64
	VersionData *permLayer.VersionData
}

var batchMode bool

// Messages collects completion messages (or errors) to be displayed at the end
type Messages struct {
	mux sync.Mutex
	buf []string
}

func (m *Messages) Add(message string) {
	m.mux.Lock()
	m.buf = append(m.buf, message)
	m.mux.Unlock()
}

func (m *Messages) Show() {
	fmt.Println("")
	for _, message := range m.buf {
		fmt.Println(message)
	}
}

var resultMessages Messages

func (u *Request) IsValid() (err error) {
	if !FileExists(u.FilePath) {
		err = fmt.Errorf("file %q does not exist", u.FilePath)
	} else if u.UploaderAccount == nil {
		err = errors.New("missing uploader account")
	} else if u.Encoding != EncodingMxor && u.NumTotal < u.NumRequired {
		err = errors.New("total shards cannot be smaller than required shards")
	}
	return
}

// Upload uploads the file (sharded or not)
// Returned price unit depends on the blockchain
// Eth=Wei, Neo=CGas*(10^8) The smallest unit of CGAS is 0.00000001
func (u *Request) Upload() (downloadUrl string, price int64, err error) {
	batchMode = u.Batch
	err = u.IsValid()
	if err != nil {
		return
	}

	numRequired := 0
	numTotal := 0
	if u.Encoding == EncodingMxor {
		numRequired = shards.BOMxorNumRequired
		numTotal = shards.BOMxorNumTotal
	} else if u.Encoding == EncodingNone {
		numTotal = 1
	} else {
		numRequired = u.NumRequired
		numTotal = u.NumTotal
	}

	var sps StorageProviders
	if BuildConfig == Debug {
		sps, err = GetUploadSpsLocal(u.UploaderAccount)
	} else {
		sps, err = GetUploadSps(numTotal, u.UploaderAccount)
	}
	if err != nil {
		return
	}

	if sps.Num() == 0 {
		err = fmt.Errorf("could not find any storage providers")
		return
	}
	a := &ArchonUrl{
		Permission: u.UploaderAccount.Permission(),
		Needed:     numRequired,
		Total:      numTotal,
	}
	if !u.HashUrl {
		cloudPath := path.Join(u.CloudDir, filepath.Base(u.FilePath))
		userName, _ := u.UploaderAccount.GetUserName()
		a.Username = userName
		a.Path = strings.ReplaceAll(cloudPath, "\\", "/")
	}
	versionData, err := u.UploaderAccount.NewVersionData()
	if err != nil {
		return
	}
	u.VersionData = versionData
	if u.Encoding != EncodingNone {
		// Shards
		price, err = u.shardedUpload(a, sps)
		resultMessages.Show()
	} else {
		// Whole file
		price, err = u.wholeFileUpload(a, sps)
	}
	if err == nil {
		// Managed to upload OK
		downloadUrl = a.String()
	}
	return
}

func startingUploadMessage(sps StorageProviders) {
	hosts := sps.Hosts()
	distinct := make(map[string]bool)
	for _, h := range hosts {
		distinct[h] = true
	}
	dh := StringKeysOf(distinct)
	fmt.Printf("Attempting upload to: %s\n", strings.Join(dh, ", "))
}

func query(name, value string) string {
	return fmt.Sprintf("%s=%s", name, value)
}

func (u *Request) TargetUrl(spUrl, txid string) string {
	turl := fmt.Sprintf("%s%s?%s&%s",
		spUrl, UploadEndpoint,
		query(TransactionHashQuery, txid),
		query(OverwriteQuery, strconv.FormatBool(u.Overwrite)),
	)
	if u.UploaderAccount != nil {
		turl += "&" + query(ChainQuery, string(u.UploaderAccount.Permission()))
	}
	if u.HashUrl {
		turl += "&" + query(HashUrlQuery, "true")
	} else if u.CloudDir != "" {
		turl += "&" + query(CloudDir, url.QueryEscape(u.CloudDir))
	}
	return turl
}
func (u *Request) UploaderPath() string {
	uploaderPath := strings.TrimLeft(u.FilePath, filepath.VolumeName(u.FilePath))
	uploaderPath = strings.ReplaceAll(uploaderPath, `\`, `/`)
	return uploaderPath
}
