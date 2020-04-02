// common includes definition and interfaces common to several Archon Go packages
package common

import (
	"encoding/json"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	Kilo = 1000
	Mega = Kilo*1000
	Giga = Mega*1000
	Quintillion = Giga*Giga	// 10^18
)

const (
	ArcProtocol = "arc"
	ShardFileSuffix = "afs"
	HashFileSuffix  = "afh"
	WholeFileSuffix = "af"

	ContainsEndpoint   = "/contains"
	RetrieveEndpoint   = "/retrieve"
	SpProfilesEndpoint = "/spprofiles"
	StatsEndpoint      = "/stats"
	UploadEndpoint     = "/upload"
	DownloadEndpoint   = "/download"
)

// Keys and queries
const (
	//--------- for upload --------------
	UploadFileKey = "uploadFile"	// the key in form-data for the file path
	TransactionHashQuery = "txHash"
	OverwriteQuery = "overwrite"
	HashUrlQuery = "hashUrl"	// request for hash URL (rather than named)
	ChainQuery = "chain"	// Ethereum or Neo
	CloudDir = "cloudDir"

	//--------- for download ----------------
	ShardIdxQuery = "shardIdx"
	ArchonUrlQuery = "archonUrl"
)

const ArchonSignatureLen = ecrypto.SignatureLength //65
type ArchonSignature [ArchonSignatureLen]byte

func NewArchonSignature(sig []byte) *ArchonSignature {
	var asig ArchonSignature
	copy(asig[:], sig)
	return &asig
}

func (a *ArchonSignature) String() string {
	return BytesToString((*a)[:])
}

type ContainsEpResponse struct {
	ShardIdx []int	`json:"shards"`
}

func NewContainsEpResponse(jsonData []byte) (*ContainsEpResponse, error) {
	containsResp := ContainsEpResponse{}
	err := json.Unmarshal(jsonData, &containsResp)
	return &containsResp, err
}

func (c *ContainsEpResponse) String() string {
	return ToJsonString(c)
}

type RetrieveResponse struct {
	ArchonUrl string		`json:"archon_url"`
	// Map from shard index to urls storing it
	Urls map[int][]string	`json:"urls"`
}

func NewRetrieveResponse(jsonData []byte) (*RetrieveResponse, error) {
	r := new(RetrieveResponse)
	err := json.Unmarshal(jsonData, r)
	return r, err
}

func (r *RetrieveResponse) String() string {
	return ToJsonString(r)
}

/*
type UploadUrlsResponse struct {
	// Map from blockchain address to upload URLs
	UploadUrls map[string][]string	`json:"upload_urls"`
}

func NewUploadUrlsResponse(jsonData []byte) (*UploadUrlsResponse, error) {
	r := new(UploadUrlsResponse)
	err := json.Unmarshal(jsonData, r)
	return r, err
}

func (r *UploadUrlsResponse) String() string {
	return ToJsonString(r)
}
*/

type SpProfilesResponse struct {
	Layer	string  `json:"layer"`
	Sps		[]SpProfile	`json:"sps"`
}

func NewSpProfilesResponse(jsonData string) (*SpProfilesResponse, error) {
	r := new(SpProfilesResponse)
	err := json.Unmarshal([]byte(jsonData), r)
	return r, err
}

func (r *SpProfilesResponse) String() string {
	return ToJsonString(r)
}
