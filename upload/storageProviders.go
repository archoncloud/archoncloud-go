package upload

import (
	"fmt"
	"github.com/gofrs/flock"
	"github.com/itsmeknt/archoncloud-go/account"
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/itsmeknt/archoncloud-go/interfaces"
	"github.com/itsmeknt/archoncloud-go/networking/archon-dht/dht_permission_layer"
	"os"
	"time"
)

/*
func getDownloadUrls(seedUrls []string, aUrl *ArchonUrl) (map[int][]string, error) {
	for _, b := range seedUrls {
		contents, err := GetFromSP(b, RetrieveEndpoint, "url="+aUrl.String())
		if err == nil {
			r, err := NewRetrieveResponse([]byte(contents));
			if err == nil {
				return r.Urls, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find SPs storing %s", aUrl)
}
*/

// currentSPs returns collection os sps retrieved from DHT
func currentSPs(acc interfaces.IAccount) (storageProviders StorageProviders, err error) {
	p := account.PermLayerID(acc)
	// there is one cache per permission layer
	cachePath := DefaultToExecutable("sps." + string(p) + ".cache")
	// Lock access to cache file since it can be executed from different processes
	lockPath := cachePath + ".lock"
	fileLock := flock.New(lockPath)
	lockStart := time.Now()
	for {
		locked, lerr := fileLock.TryLock()
		if lerr != nil {
			err = lerr
			return
		}
		if locked {break}
		if time.Since(lockStart) > 30*time.Second {return}
	}
	defer fileLock.Unlock()

	// Check if cache needs refresh
	file, err := os.Stat(cachePath)
	const hoursBetweenRefresh = 2
	if err != nil || time.Since(file.ModTime()).Hours() > hoursBetweenRefresh {
		err = refreshCache(p, cachePath)
		if err != nil {return}
	}
	var sps StorageProviders
	err = GetConfiguration(&sps, cachePath)
	if err != nil {return}
	if sps.Num() > 0 {
		storageProviders = sps;
		return
	}
	// Try to refresh
	err = refreshCache(p, cachePath)
	if err != nil {return}
	err = GetConfiguration(&sps, cachePath)
	if sps.Num() == 0 {
		err = fmt.Errorf("could not find any storage providers to upload to")
		return
	}
	storageProviders = sps
	return
}

func refreshCache(p dht_permission_layer.PermissionLayerID, cachePath string) (err error) {
	// Map from address to profile
	m := make(map[string]SpProfile)
	for _, seed := range GetAllSeedUrls() {
		contents, err := GetFromSP(seed, SpProfilesEndpoint, "layer="+string(p), 5*time.Second)
		prev := len(m)
		if err == nil {
			r, err := NewSpProfilesResponse(contents);
			if err == nil {
				for _, sp := range r.Sps {
					if sp.IsValid() {
						m[sp.Address] = sp
					}
				}
			}
		}
		if prev > 0 && prev == len(m) {
			// No new data added
			break
		}
	}
	retrieved := len(m)
	sps := NewStorageProviders(retrieved)
	if retrieved > 0 {
		i := 0
		for _, sp := range m {
			sps.Set(i, &sp)
			i++
		}
	}
	err = SaveConfiguration(&sps, cachePath)
	return
}

// GetUploadSps returns up to needed SP profiles, after marketplace filtering
func GetUploadSps(needed int, acc interfaces.IAccount) (sps StorageProviders, err error) {
	// Get all the known SPs for this acc
	sps, err = currentSPs(acc)
	// Marketplace mechanism
	// TODO: add other criteria, region, SLA etc.
	if err == nil {
		if sps.Num() > needed {
			// Can pick and choose
			granularity := uint64(acc.HundredthOfCent())
			sps.SortByMinAsk(granularity)
			// Refine and randomize
			// Pick all with a price equal to the largest amongst needed
			last := needed-1
			high := RoundUp(uint64(sps.Get(last).MinAskPrice), granularity)
			for ; last+1 < sps.Num(); last++ {
				if uint64(sps.Get(last).MinAskPrice) > high {
					break;
				}
			}
			sps.KeepOnly(last)
			// Get random needed
			sps = sps.PickRandom(needed)
		}
	}
	return
}

// GetUploadSpsLocal is for debugging only, using an SP on localhost
func GetUploadSpsLocal(u *Request) (sps StorageProviders, err error) {
	sps = NewStorageProviders(0)
	sp := SpProfile{
		Urls:               Urls{
			Host:      "localhost",
			HttpPort:  "9000",
			HttpsPort: "",
		},
		AvailableGigaBytes: 0,
		PledgedGigaBytes:   11.1,
		NodeId:             "?",
	}
	// the min Ask is per MByte. Aiming for 0.04c / GByte -> 0.00004c / MByte
	sp.MinAskPrice = u.UploaderAccount.HundredthOfCent()*4 / 1000
	if u.UploaderAccount.GetAccountType() == interfaces.EthAccountType {
		sp.Address = "0x8cf8611cfa222ffbcf6a762734aab200f40a4970"
	} else {
		// in Gas per MByte
		sp.Address = "Aey8DHNKSQCrHTpVRK32fLsJtXkAtdzSbJ" //dev2
	}
	sps.Add(&sp)
	return
}
