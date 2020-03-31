package download

import (
	"bytes"
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/shards"
	"github.com/pkg/errors"
	"io"
	fp "path/filepath"
	"time"
)

/*
for the Download client, command line usage:
1. ./archonDownload <archonUrl_or_archonHash>, or
2. ./archonDownload --ethereumWallet=... <archonUrl_or_archonHash>
-it then interactively asks for the ethereum wallet password to unlock it
Option 2 will not be used currently, but in the future we will use the ethereum wallet
The way the archon download function works:
1. on initialization, loads list of SPs from cache. If cache is empty or expired, downloads list of SPs and saves them in local cache (expires daily or with command line --purge-cache)
2. compute totalShards, which is 1 if the file is unencoded or 6 if it is mxor26 encoded
3. compute shardIdxQueue, which is shuffle([1, ..., totalShards])
4. compute numRequiredShards, which is 1 if the file is unencoded or 2 if it is mxor26 encoded
5. for (int i = 0; i < numRequiredShards; i++) {
multithreaded_download_routine();
}
6. the multithreaded_download_routine() pops a shardIdx from the shardIdxQueue. if the queue is empty, end the download routine with an error message "cannot download enough shards to reconstruct. downloaded shards: X, required shards: Y"
7. given a shardIdx, download shard from listOfSps[shardIdx % listOfSps.length] <--- currently either the SP has it, or fails. later on, when we implement DHT, the SP would redirect to the correct peer that does have it
*/

type Request struct {
	ArchonUrl		*ArchonUrl
	DownloadFolder	string
	DownloadFileName string
	Overwrite		bool
	PreferHttp		bool
	Batch			bool
}

type downloadOpResult struct {
	err		error
	shardIx	int
	shardBytes []byte
}

// Download generates original file contents into DownloadFolder
func (req *Request) Download() (err error) {
	fmt.Printf("Downloading %q\n", req.ArchonUrl)

	downloadPath := req.DownloadFolder
	if req.DownloadFileName != "" {
		downloadPath = fp.Join(downloadPath, req.DownloadFileName)
	} else {
		downloadPath = fp.Join(downloadPath, fp.Base(req.ArchonUrl.Path))
	}
	if !req.Overwrite && FileExists(downloadPath) {
		return fmt.Errorf("file %q already exists", downloadPath)
	}

	downloadMap, err := GetDownloadUrls(req.ArchonUrl); if err != nil {return}
	urlMap := getUrlMap(downloadMap, req.PreferHttp)
	if len(urlMap) == 0 {
		err = fmt.Errorf("could not find SPs storing %s", req.ArchonUrl)
		return
	}

	fmt.Println("Downloading...")
	if req.ArchonUrl.IsWholeFile() {
		err = downloadWhole(req.ArchonUrl, downloadPath, urlMap)
	} else {
		err = downloadSharded(req.ArchonUrl, downloadPath, urlMap)
	}
	if err == nil {
		fmt.Printf("Downloaded to %q (%s)\n", downloadPath, FileSizeString(downloadPath))
	}
	return
}

// downloadWhole downloads a non-sharded file
func downloadWhole(aUrl *ArchonUrl, downloadPath string, urlMap map[int]string) error {
	// Find the SP that has it
	var spUrl string
	switch len(urlMap) {
	case 0:
		return errors.New("none of the SPs has this file")
	case 1:
		//TODO: maybe try more than the first
		spUrl = urlMap[0]
	default:
		return errors.New("expected only one entry")
	}

	file, err := CreateFile(downloadPath); if err != nil {return err}
	defer file.Close()

	downloadUrl := fmt.Sprintf("%s%s", spUrl, aUrl.DownloadUrl(-1))
	pipeReader, pipeWriter := io.Pipe()
	var writeError error
	go func() {
		// close the writer, so the reader knows there's no more data
		defer pipeWriter.Close()
		writeError = DownloadFile(pipeWriter, downloadUrl)
	}()

	_, err = io.Copy(file, pipeReader)
	if err != nil {return err}
	if writeError != nil {return writeError}
	return nil
}

func downloadSharded(a *ArchonUrl, downloadPath string, urlMap map[int]string) (err error) {
	numShardsFound := len(urlMap)
	if numShardsFound < a.Needed {
		if numShardsFound == 0 {
			return fmt.Errorf("No shards corresponding to this URL were found. If you just uploaded, you may need to wait few minutes.")
		}
		return fmt.Errorf("needed %d shards, only found %d", a.Needed, numShardsFound)
	}

	// Download all needed shards
	downloadOps := make(chan downloadOpResult, numShardsFound)
	defer close(downloadOps)
	for ix, sp := range urlMap {
		downloadUrl := fmt.Sprintf("%s/%s", sp, a.DownloadUrl(ix))
		go downloadShard(ix, downloadUrl, downloadOps)
	}

	// Wait for completion and store. Each goroutine should complete either with error or success
	shardMap := make(map[int][]byte)
	for k := numShardsFound; k > 0; k-- {
		res := <- downloadOps
		if res.err != nil {
			fmt.Printf("Error for shard %d: %v", res.shardIx, res.err)
		} else {
			shardMap[res.shardIx] = res.shardBytes
		}
	}

	downloadedShards := len(shardMap)
	if downloadedShards < a.Needed {
		// TODO: if we do not get all shards (maybe some SPs died) need to try again
		return fmt.Errorf("%d shards downloaded, needed %d", downloadedShards, a.Needed)
	}

	// Find out the type of the shard (from first one), generate container
	// and store shards in container
	var s *shards.ShardsContainer
	for ix, data := range shardMap {
		if s == nil {
			s, err = shards.NewShardsContainer(data)
			if err != nil {return}
		}
		sh, err2 := shards.NewShardFromSP(data); if err2 != nil {err = err2; return}
		err = s.SetShard(ix, sh); if err != nil {return}
	}

	// Now decode and store in file
	file, err := CreateFile(downloadPath)
	defer file.Close()
	if err == nil {
		// Decode generates a file container and writes the original data
		err = s.Decode(s, file)
	}
	return
}

// downloadShard downloads the whole shard container
func downloadShard(ix int, downloadUrl string, ch chan<- downloadOpResult ) {
	var b bytes.Buffer
	err := DownloadFile(&b, downloadUrl)
	res := downloadOpResult{shardIx: ix}
	if err != nil {
		res.err = err
	} else {
		res.shardBytes = b.Bytes()
	}
	ch <- res
}

func getContains(spUrl string, aUrl *ArchonUrl) (*ContainsEpResponse, error) {
	contents, err := GetFromSP(spUrl, ContainsEndpoint, "url="+aUrl.String(), 2*time.Second)
	if err != nil {return nil, err}

	return NewContainsEpResponse([]byte(contents))
}

func getUrlMap(downloadMap map[int][]string, preferHttp bool) map[int]string {
	urlMap := make(map[int]string)
	for shardIx, urls := range downloadMap {
		for _, urlList := range urls {
			if urlList != "" {
				u, err := NewUrls(urlList)
				if err == nil {
					urlMap[shardIx] = u.Url(preferHttp)
				}
			}
		}
	}
	return urlMap
}
