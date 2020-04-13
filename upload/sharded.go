package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/archoncloud/archoncloud-go/account"
	. "github.com/archoncloud/archoncloud-go/common"
	ifs "github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/archoncloud/archoncloud-go/shards"
	"github.com/pkg/errors"
)

// the process of getting one shard from spAddress
type shardUpload struct {
	shards     *shards.ShardsContainer
	shardIndex int
	spProfile  *SpProfile
	uploadUrl  string
	err        error
}

func (su *shardUpload) PickUrl(preferHttp bool) {
	su.uploadUrl = su.spProfile.Url(preferHttp)
}

func (su *shardUpload) GetShard() *shards.Shard {
	return su.shards.GetShard(su.shardIndex)
}

func (u *Request) generateFileContainer() (fileContainer *shards.FileContainer, err error) {
	file, err := os.Open(u.FilePath)
	if err != nil {
		return
	}
	defer file.Close()

	var s *shards.ShardsContainer
	switch u.Encoding {
	case EncodingMxor:
		s = shards.NewBOMxor()
	case EncodingRSa:
		s = shards.NewAors(u.NumTotal, u.NumRequired)
	case EncodingRSb:
		s = shards.NewBors(u.NumTotal, u.NumRequired)
	default:
		err = fmt.Errorf("unknown encoding: %s", u.Encoding)
		return
	}
	fmt.Println("Generating container and encoding shards...")
	return shards.NewFileContainer(s, u.VersionData, file, u.UploaderAccount)
}

// shardedUpload does the upload of shards
func (u *Request) shardedUpload(a *ArchonUrl, sps StorageProviders) (price int64, err error) {
	fileContainer, err := u.generateFileContainer()
	if err != nil {
		return
	}
	if a.IsHash() {
		a.Path = fileContainer.Shards.GetOriginDataHashString()
	}
	// New proposed upload transaction
	var txIds map[string]string
	switch u.UploaderAccount.GetAccountType() {
	case ifs.EthAccountType:
		var txId string
		txId, price, err = account.ProposeUpload(u.UploaderAccount, fileContainer, fileContainer.Shards, a, sps, u.MaxPayment)
		if err == nil {
			// Upload all the shards
			for _, sp := range sps {
				txIds[sp.Address] = txId
			}
			err = u.uploadNeededShards(txIds, fileContainer.Shards, sps)
			if err == nil {
				fmt.Println("Upload transaction ID:", txId)
			}
		}
	case ifs.NeoAccountType:
		neoAcc := u.UploaderAccount.(*account.NeoAccount)
		txIds, price, err = neoAcc.ProposeUpload(fileContainer, fileContainer.Shards, a, sps, u.MaxPayment)
		if err != nil {
			return
		}
		err = u.uploadNeededShards(txIds, fileContainer.Shards, sps)
		if err == nil {
			fmt.Println("Upload transaction IDs:")
			for _, tx := range txIds {
				fmt.Println("   ", tx)
			}
		}
	default:
		err = fmt.Errorf("unknown account type")
	}
	return
}

// uploadNeededShards uploads the shards in parallel to sps
func (u *Request) uploadNeededShards(txIds map[string]string, s *shards.ShardsContainer, sps StorageProviders) (err error) {
	n := s.GetNumShards()
	shardsToDo := make([]int, n)
	for i := 0; i < s.GetNumShards(); i++ {
		shardsToDo[i] = i
	}
	// All shards have same length
	shardLen := s.GetShard(0).Len()
	totalLen := uint64(shardLen) * uint64(n)
	fmt.Printf("Starting upload of %s\n", NumBytesDisplayString(totalLen))
	bp := NewByteProgress("Uploading", totalLen)
	startingUploadMessage(sps)
	for len(shardsToDo) > 0 {
		if sps.Num() == 0 {
			// No more SPs to try, but we still have shards. We must have at least one error reported
			err = errors.New("could not upload all shards")
			break
		}

		uploads := getShardUploads(s, shardsToDo, sps, 7)

		// Uploads in parallel
		todo := len(uploads)
		uchan := make(chan *shardUpload, todo)
		for _, shu := range uploads {
			go u.postShard(shu, txIds[shu.spProfile.Address], bp, uchan, u.PreferHttp)
		}

		// Wait for this set to respond
		for ; todo > 0; todo-- {
			resp := <-uchan
			if resp.err != nil {
				errMsg := resp.err.Error()
				errMsg = strings.TrimRight(errMsg, "\n")
				errMsg = strings.TrimLeft(errMsg, "Error: ")
				resultMessages.Add(fmt.Sprintf("Error %s: %s", resp.spProfile.Urls.Host, errMsg))
				// Don't try this SP again since it has failed
				sps.Remove(func(sp *SpProfile) bool { return sp.Address == resp.spProfile.Address })
			} else {
				// This shard is done, yay!
				resultMessages.Add(fmt.Sprintf("Shard %d uploaded to %s", resp.shardIndex, resp.uploadUrl))
				shardsToDo = EraseInt(shardsToDo, resp.shardIndex)
			}
		}
	}
	bp.End()
	return
}

func getShardUploads(s *shards.ShardsContainer, shardsToDo []int, sps StorageProviders, maxAtOnce int) []*shardUpload {
	n := Min(len(shardsToDo), maxAtOnce)
	uploads := make([]*shardUpload, n)
	numSps := sps.Num()
	for ix, shardIndex := range shardsToDo {
		if ix >= n {
			break
		}
		uploads[ix] = &shardUpload{
			shards:     s,
			shardIndex: shardIndex,
			spProfile:  sps.Get(ix % numSps),
			err:        nil,
		}
	}
	return uploads
}

// postShard posts a shard to a given SP
func (u *Request) postShard(su *shardUpload, txid string, bp *ByteProgress, uchan chan *shardUpload, preferHttp bool) {
	// Use a pipe so we don't need the whole shard in memory
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	var writeErr error
	go func() {
		defer w.Close()
		defer m.Close()
		part, writeErr := m.CreateFormFile(UploadFileKey, u.FilePath)
		if writeErr != nil {
			return
		}
		mm := multipart.NewWriter(part)
		sVersionData, writeErr := json.Marshal(u.VersionData)
		if writeErr != nil {
			return
		}
		writeErr = mm.WriteField("UploadVersion", string(sVersionData))
		if writeErr != nil {
			return
		}
		shard := su.GetShard()
		writeErr = shard.WriteShardContainer(part, u.UploaderAccount)
	}()

	success := false
	errorList := make([]error, 0)
	usedUrls := make(map[string]bool)
	// Can try both http and https
	for k := 0; k < 2; k++ {
		su.PickUrl(preferHttp)
		if su.uploadUrl != "" {
			if _, ok := usedUrls[su.uploadUrl]; ok {
				// Already tried
				continue
			}
			usedUrls[su.uploadUrl] = true
			targetUrl := u.TargetUrl(su.uploadUrl, txid)
			uplErr := PostFromReaderWithProgress(targetUrl, r, m.FormDataContentType(), bp)
			if uplErr == nil {
				uplErr = writeErr
			}
			if uplErr == nil {
				success = true
				break
			}
			errorList = append(errorList, uplErr)
		}
		preferHttp = !preferHttp
	}
	if !success {
		if len(errorList) == 0 {
			su.err = fmt.Errorf("no urls available for %s", su.spProfile.Host())
		} else {
			su.err = errorList[0]
		}
	}
	uchan <- su
}
