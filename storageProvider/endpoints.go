package storageProvider

import (
	"fmt"
	"github.com/archoncloud/archon-dht/permission_layer"
	"github.com/archoncloud/archoncloud-go/account"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"net/http"
	fp "path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func infoHandler(w http.ResponseWriter, r *http.Request) {
	zone, offset := time.Now().Zone()
	httpInfo2(w, r, fmt.Sprintf("Archon Storage Provider V%s\nUp since %v\nTime zone: %s (%d)\n%s %s",
		SPVersion, StartTime.Format(time.RFC1123),
		zone, offset,
		runtime.GOOS, runtime.GOARCH))
}

type StatsResponse struct {
	TotalStored		string
	Eth struct {
		TotalPledged string           `json:",omitempty"`
	}`json:",omitempty"`
	Neo struct {
		TotalPledged string           `json:",omitempty"`
	}`json:",omitempty"`
}

// statsHandler implements the "/stats" endpoint. Storage related info
func statsHandler(w http.ResponseWriter, r *http.Request) {
	resp := StatsResponse{}
	LogInfo.Printf("\"stats\" request from = %q\n", r.URL.Host)
	storedBytes, _ := DirSize(GetHashesFolder())
	n, _ := DirSize(GetShardsFolder())
	storedBytes += n
	resp.TotalStored = humanize.Bytes(uint64(storedBytes))
	for _, acc := range []interfaces.IAccount{SPAccount.Eth,SPAccount.Neo} {
		if acc != nil {
			sps, err := GetSPProfiles(account.PermLayerID(acc))
			if err == nil {
				sp := sps.GetOfAddress(acc.AddressString())
				if sp != nil {
					pledged := humanize.Bytes(uint64(sp.PledgedGigaBytes * humanize.GiByte))
					if account.IsEth(acc) {
						resp.Eth.TotalPledged = pledged
					} else {
						resp.Neo.TotalPledged = pledged
					}
				}
			}
		}
	}
	httpInfo(w, r, ToJsonString(resp))
}

// showLogHandler implements "/log" endpoint
func showLogHandler(w http.ResponseWriter, r *http.Request) {
	LogInfo.Printf("%s\n", requestInfo(r))
	WriteLastLines(w, GetLogFilePath(), 60)
}

// retrieveHandler implements the "/retrieve" endpoint
// for download info needed by clients
func retrieveHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("url")
	LogTrace.Printf("%s: for Archon URL %q\n", requestInfo(r), q)
	aUrl, err := NewArchonUrl(q)
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}
	// Try to get info on as many shards as possible, although we may not need them all
	totalShards := aUrl.ShardPaths()
	response := RetrieveResponse{
		aUrl.String(),
		make(map[int][]string),
	}
	timeout := 8*time.Second
	start := time.Now()
	// TODO: maybe do this in parallel
	for ix, sh := range totalShards {
		since := time.Since(start)
		if since > timeout {
			httpErr(w, r, errors.New("timeout"), http.StatusRequestTimeout )
		}
		urls := GetDownloadUrlsForShard(sh, timeout-since)
		if len(urls) > 0 {
			response.Urls[ix] = urls
		}
	}
	httpInfo(w, r, response.String())
}

func getPermLayer(r *http.Request) permission_layer.PermissionLayerID {
	layer := strings.ToUpper(r.URL.Query().Get("layer"))
	pl := permission_layer.PermissionLayerID(layer)
	switch pl {
	case permission_layer.EthPermissionId:
	case permission_layer.NeoPermissionId:
	case permission_layer.NotPermissionId:
	default: pl = permission_layer.EthPermissionId
	}
	return pl;
}

// spProfilesHandler implements the "/spprofiles" endpoint (SpProfilesEndpoint)
func spProfilesHandler(w http.ResponseWriter, r *http.Request) {
	//?layer=ETH
	pl := getPermLayer(r)
	response := new(SpProfilesResponse)
	response.Layer = string(pl)
	sps, err := GetSPProfiles(pl)
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}
	for _, sp := range sps {
		response.Sps = append(response.Sps,sp)
	}
	httpDebug(w, r, response.String())
}

// containsHandler implements the "/contains" endpoint
func containsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("url")
	aUrl, err := NewArchonUrl(q)
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}

	var pattern string
	var indexWildCard string
	if !aUrl.IsWholeFile() {
		indexWildCard = ".*."
	} else {
		indexWildCard = "."
	}

	if aUrl.IsHash() {
		pattern = InHashesFolder(aUrl.Path + indexWildCard + HashFileSuffix)
	} else {
		var suffix string
		if aUrl.IsWholeFile() {
			suffix = WholeFileSuffix
		} else {
			suffix = ShardFileSuffix
		}
		pattern = InShardsFolder(aUrl.Username + aUrl.Path + indexWildCard + suffix)
	}

	var response = ContainsEpResponse{}
	shards, err := fp.Glob(pattern)
	if err == nil {
		if aUrl.IsWholeFile() {
			if len(shards) > 0 {
				response.ShardIdx = append(response.ShardIdx, -1)
			}
		} else {
			// Extract indices
			for _, f := range shards {
				parts := strings.Split(f, ".")
				l := len(parts)
				if l > 2 {
					if index, err := strconv.Atoi(parts[l-2]); err == nil {
						response.ShardIdx = append(response.ShardIdx, index)
					}
				}
			}
		}
	}
	httpInfo(w, r, response.String())
}
