package storageProvider

import (
	"fmt"
	dht "github.com/archoncloud/archon-dht/archon"
	dhtp "github.com/archoncloud/archon-dht/dht_permission_layers"
	"github.com/archoncloud/archon-dht/permission_layer"
	"github.com/archoncloud/archoncloud-ethereum/client_utils"
	"github.com/archoncloud/archoncloud-go/account"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/dustin/go-humanize"
	"sort"
	"time"
)

var dhtInstance *dht.ArchonDHTs

// GetDownloadUrlsForShard returns download URLs that have store this shard. Ignores errors
func GetDownloadUrlsForShard(shard string, timeout time.Duration) (mergedUrls []string) {
	u := make(map[string]bool)
	uMap, err := dhtInstance.GetUrlsOfNodesHoldingKeysFromAllLayers([]string{shard}, timeout)
	if err == nil {
		for _, urls := range uMap {
			for _, url := range urls {
				u[url] = true
			}
		}
		for url, _ := range u {
			mergedUrls = append(mergedUrls,url)
		}
	}
	LogDebug.Printf("GetDownloadUrlsForShard %s returns %v", shard, mergedUrls)
	return
}

func GetSPProfiles(layer permission_layer.PermissionLayerID) (StorageProviders, error) {
	sps := NewStorageProviders(0)
	profiles, err := dhtInstance.GetArchonSPProfilesForMarketplace(layer)
	if err != nil {return sps, err}

	// First make sure we have all the Urls
	nodesWithNoUrl := make([]string,0)
	for _, sp := range profiles {
		if sp.Url == "" {
			nodesWithNoUrl = append(nodesWithNoUrl,sp.NodeID)
		}
	}

	urlMap := make(map[string]string)
	if len(nodesWithNoUrl) != 0 {
		// Ask for the missing urls
		LogDebug.Printf("Calling GetUrls for %d nodes", len(nodesWithNoUrl))
		urlMap, err = dhtInstance.GetUrls(nodesWithNoUrl, layer, 2*time.Second)
		LogDebug.Println("GetUrls returned")
		if err != nil {
			return sps, err
		}
	}

	// Now fill in all the info
	for _, sp := range profiles {
		prof := SpProfile{}
		prof.NodeId = sp.NodeID
		prof.Address = string(sp.Address)
		prof.MinAskPrice = int64(sp.MinAskPrice)
		av := float64(sp.RemainingStorage)
		prof.AvailableGigaBytes = av / humanize.GByte
		prof.PledgedGigaBytes = float64(sp.PledgedStorage) / humanize.GByte
		url := sp.Url
		if url == "" {
			// Get from map
			url = urlMap[sp.NodeID]
		}
		urls, err := NewUrls(url)
		if err != nil {
			// Ignore for now
			continue
		}
		prof.Urls = *urls
		sps.Add(&prof)
	}
	return sps, nil
}

func AnnounceToDht(shard, layerId string) (err error) {
	layer := dhtp.NewPermissionLayer(layerId)
	if layer == nil {
		err = fmt.Errorf("invalid layer %q", layerId)
		return
	}
	v, err := layer.NewVersionData()
	if err != nil {return}
	LogDebug.Printf("Calling Stored for %s, args: %q %v\n", layerId, shard, v)
	err = dhtInstance.Stored(shard, v)
	return
}

// showInfo just displays marketplace info about the registered SPs and this SP
func showInfo() {
	if SPAccount.Eth != nil {
		sps, err := GetSPProfiles(permission_layer.EthPermissionId)
		Abort(err)
		asks := make([]int64, 0)
		for _, sp := range sps {
			asks = append(asks, int64(sp.MinAskPrice))
		}
		l := len(asks)
		if l == 0 {
			fmt.Println("There are no SP accounts registered")
		} else {
			sort.Slice(asks, func(i, j int) bool {
				return asks[i] < asks[j]
			})
			fmt.Printf("\n\n\nFor Ethereum In Wei per Byte\n")
			fmt.Printf("%d storage providers registered:\n", l)
			fmt.Printf("min=%s median=%s max=%s\n",
				account.WeiPerMByteString(asks[0]),
				account.WeiPerMByteString(asks[l/2]),
				account.WeiPerMByteString(asks[l-1]))

			ourSP := sps.GetOfAddress(SPAccount.Eth.AddressString())
			if ourSP == nil {
				fmt.Println("this SP is not in the registered list")
			} else {
				fmt.Printf("this SP=%s\n", account.WeiPerMByteString(ourSP.MinAskPrice))
				balance, err := client_utils.GetEarnings(*SPAccount.Eth.GetEthAddress())
				if err != nil {
					fmt.Println("Can't get earnings of this SP")
				} else {
					fmt.Printf("this SP earnings=%s\n", account.WeiString(balance.Int64()))
				}
			}
		}
	}
}
