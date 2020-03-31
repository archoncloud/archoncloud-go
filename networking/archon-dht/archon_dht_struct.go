package archon_dht

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/itsmeknt/archoncloud-go/common"

	"github.com/ipfs/go-cid"

	. "github.com/itsmeknt/archoncloud-go/blockchainAPI/registered_sp"
	permLayer "github.com/itsmeknt/archoncloud-go/networking/archon-dht/dht_permission_layer"

	dht "github.com/itsmeknt/archoncloud-go/networking/archon-dht/mods/kad-dht-mod"

	"github.com/libp2p/go-libp2p-core/peer"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	mh "github.com/multiformats/go-multihash"
)

type ArchonDHT struct {
	PermissionLayerId permLayer.PermissionLayerID
	routedHost        *rhost.RoutedHost
	dHT               *dht.IpfsDHT
	Config            DHTConnectionConfig
}

type UrlArray []string
type PermissionLayer2ArchonDHT map[permLayer.PermissionLayerID]*ArchonDHT
type PermissionLayer2UrlArray map[permLayer.PermissionLayerID]UrlArray

type ArchonDHTs struct {
	Layers PermissionLayer2ArchonDHT
}

type UrlsStruct struct {
	Urls string `json:download_urls`
}

type UrlsVersionedStruct struct {
	Urls       string                `json:download_urls`
	Versioning permLayer.VersionData `json:versioning`
}

func (a *ArchonDHTs) Init() {
	a.Layers = make(PermissionLayer2ArchonDHT)
}

func (a *ArchonDHTs) AddLayer(permissionLayerId permLayer.PermissionLayerID, rh *rhost.RoutedHost, d *dht.IpfsDHT, c DHTConnectionConfig) {
	archonDHT := new(ArchonDHT)
	archonDHT.PermissionLayerId = permissionLayerId
	archonDHT.routedHost = rh
	archonDHT.dHT = d
	archonDHT.Config = c
	a.Layers[permissionLayerId] = archonDHT
}

func (a ArchonDHTs) Stored(key string, versionData *permLayer.VersionData) error {
	common.LogDebug.Println("Stored ", key)
	keyAsCid, err := StringToCid(key)
	if err == nil {
		err = a.putValueVersioned("/archondl/", keyAsCid, *versionData)
	}
	common.LogDebug.Println("Stored returns ", err)
	return err
}

func (a *ArchonDHTs) GetArchonDHT(id permLayer.PermissionLayerID) *ArchonDHT {
	return a.Layers[id]
}

func StringsToCids(keys []string) ([]cid.Cid, error) {
	var ret []cid.Cid
	for i := 0; i < len(keys); i++ {
		cid, err := StringToCid(keys[i])
		if err != nil {
			return nil, err
		}
		ret = append(ret, cid)
	}
	return ret, nil
}

func StringToCid(key string) (cid.Cid, error) {
	h := sha256.New()
	h.Write([]byte(key))
	hashed := h.Sum(nil)
	multihash, err := mh.Encode([]byte(hashed), mh.SHA2_256)
	if err != nil {
		return *new(cid.Cid), err
	}
	return cid.NewCidV0(multihash), nil
}

func (a *ArchonDHTs) GetArchonSPProfilesForMarketplace(permissionLayerID permLayer.PermissionLayerID) (c []RegisteredSp, e error) {
	common.LogDebug.Println("GetArchonSPProfilesForMarketplace ", permissionLayerID)
	var ret []RegisteredSp
	var errRet string
	if permissionLayerID == "NON" {
		return ret, fmt.Errorf("error GetArchonSPProfilesForMarketplace: NON layer does not provide profiles for marketplace.")
	}
	// cacheFilepath
	cacheFilepath := ".sp_profiles_cache/" + string(permissionLayerID) + "/"
	// list all files in cacheFilepath
	fileInfo, err := ioutil.ReadDir(cacheFilepath)
	if err != nil {
		errRet += " " + err.Error()
	}
	for _, file := range fileInfo {
		filepath := cacheFilepath + file.Name()
		SpFilenames.Lock()
		if SpFilenames.M[filepath] == nil {
			SpFilenames.M[filepath] = new(permLayer.SafeFilename)
		}
		ptr := SpFilenames.M[filepath]
		SpFilenames.Unlock()
		ptr.Lock()
		defer ptr.Unlock()
		data, err := ioutil.ReadFile(filepath)
		if err != nil {
			errRet += " " + err.Error()
		}
		spProf := new(RegisteredSp)
		err = json.Unmarshal(data, &spProf)
		if err != nil {
			errRet += " " + err.Error()
		}
		if spProf.Url == "" {
			continue
		}
		ret = append(ret, *spProf)
	}
	common.LogDebug.Println("GetArchonSPProfilesForMarketplace ", ret, errRet)
	if len(errRet) > 0 {
		return ret, fmt.Errorf(errRet)
	}
	return ret, nil
}

func (a *ArchonDHTs) GetUrls(nodeIDs []string,
	permissionLayerID permLayer.PermissionLayerID,
	timeout time.Duration) (map[string]string,
	error) {
	common.LogDebug.Println("GetUrls ", nodeIDs, permissionLayerID, timeout)
	return a.Layers[permissionLayerID].getUrls(nodeIDs, timeout)
}

// each key is some pre-image of the cid mapping. depending on implementation, this can be
// an archonNamedUrl + <shardIdx>, archonHashUrl + <shardIdx>, etc
func (a *ArchonDHTs) GetUrlsOfNodesHoldingKeysFromAllLayers(keys []string, timeoutInSeconds time.Duration) (PermissionLayer2UrlArray, error) {
	common.LogDebug.Println("GetUrlsOfNodesHoldingKeysFromAllLayers ", keys, timeoutInSeconds)
	bundledMessage := make(chan bundled, len(a.Layers))
	for layerKey, v := range a.Layers {
		go func(k permLayer.PermissionLayerID, d *ArchonDHT) {
			peerAddrInfo, err := d.GetUrlsOfNodesHoldingKeys(keys, timeoutInSeconds)
			bund := bundled{PeerAddrInfo: peerAddrInfo, Error: err, Key: k}
			bundledMessage <- bund
		}(layerKey, v)
	}
	retMap := make(PermissionLayer2UrlArray)
	var retErrorString string
	for i := 0; i < len(a.Layers); i++ {
		b := <-bundledMessage
		retMap[b.Key] = b.PeerAddrInfo
		if b.Error != nil {
			retErrorString += b.Error.Error()
		}
	}
	if len(retErrorString) > 0 {
		common.LogDebug.Println("GetUrlsOfNodesHoldingKeysFromAllLayers error ", retErrorString)
		return nil, fmt.Errorf(retErrorString)
	}
	common.LogDebug.Println("GetUrlsOfNodesHoldingKeysFromAllLayers ", retMap)
	return retMap, nil
}

///////////////////////////////////////////////////////////////////////////
/// below are on single layer /////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////
/// this can be seen as the difference between ArchonDHTs and ArchonDHT ///
///////////////////////////////////////////////////////////////////////////

type NodeIDAndUrl struct { // used in getUrls
	nodeID string
	url    string
}

func (a *ArchonDHT) getUrls(nodeIDs []string,
	timeoutInSeconds time.Duration) (m map[string]string, e error) {

	cacheFilepath := ".sp_profiles_cache/" + string(a.PermissionLayerId) + "/"
	ctx, _ := context.WithTimeout(context.Background(), timeoutInSeconds)
	retMessage := make(chan map[string]string, 1)

	myNodeID := a.routedHost.ID()
	go func(nids []string, cacheFp string) {
		collectNodeIDAndUrlMessages := make(chan NodeIDAndUrl, len(nids))
		var wg sync.WaitGroup
		wg.Add(len(nids))

		for _, nid := range nids {
			go func(n, cfp string, w *sync.WaitGroup) {
				nodeIDAndUrlMessage := make(chan NodeIDAndUrl, 1)
				defer w.Done()
				go func(_n string) {
					// do stuff
					if _n == "" {
						nodeIDAndUrlMessage <- NodeIDAndUrl{nodeID: _n, url: ""}
						return
					}
					// check cache
					cacheProfileExists := false
					filepath := cfp + _n
					spDataFilepath := common.DefaultToExecutable(filepath)
					SpFilenames.Lock()
					if SpFilenames.M[spDataFilepath] == nil {
						SpFilenames.M[spDataFilepath] = new(permLayer.SafeFilename)
					}
					ptr := SpFilenames.M[spDataFilepath]
					SpFilenames.Unlock()
					ptr.Lock()
					defer ptr.Unlock()
					spProf := new(RegisteredSp)
					data, err := ioutil.ReadFile(spDataFilepath)
					if err != nil {
						cacheProfileExists = false // redundant
					}
					if len(data) > 0 {
						cacheProfileExists = true
						json_err := json.Unmarshal(data, &spProf)
						if json_err == nil {
							if spProf.Url != "" {
								nodeIDAndUrlMessage <- NodeIDAndUrl{nodeID: _n, url: spProf.Url}
								// we obtained url from cache
								return
							}
						}
					}
					// wasn't in cache so we continue
					if _n == myNodeID.Pretty() { // is it self?
						if cacheProfileExists {
							// put in cache
							updatedProf := spProf
							updatedProf.Url = a.Config.Url
							cacheUrl(updatedProf, spDataFilepath)
						}
						nodeIDAndUrlMessage <- NodeIDAndUrl{nodeID: _n, url: a.Config.Url}
						return
					}
					// since not self, retrieve url from dht
					urlFromDHT, err := a.getUrlFromDHT(ctx, spProf, cacheProfileExists, _n, spDataFilepath)
					if err == nil {
						nodeIDAndUrlMessage <- NodeIDAndUrl{nodeID: _n, url: urlFromDHT}
						return
					}
				}(n)
				select {
				case <-ctx.Done():
					return
				case nm := <-nodeIDAndUrlMessage:
					collectNodeIDAndUrlMessages <- nm
					return
				}
			}(nid, cacheFp, &wg)
		}
		wg.Wait()
		// load retMessage
		var ret map[string]string
		ret = make(map[string]string)
		lenMessage := len(collectNodeIDAndUrlMessages)
		for i := 0; i < lenMessage; i++ {
			n := <-collectNodeIDAndUrlMessages
			if n.url == "" {
				continue
			}
			ret[n.nodeID] = n.url
		}
		retMessage <- ret
		// emit searchIsComplete
	}(nodeIDs, cacheFilepath)

	select {
	case r := <-retMessage:
		return r, nil
	}
}

// a subroutine of getUrls above
func cacheUrl(updatedProf *RegisteredSp, spDataFilepath string) {
	marshalledSP, err := json.Marshal(updatedProf)
	if err != nil {
		common.LogError.Println("Caching archonSP error: ", err)
	}
	if len(marshalledSP) > 0 {
		ioutil.WriteFile(spDataFilepath, marshalledSP, 0644) // file is locked in getUrls
	} else {
		common.LogError.Println("Caching archonSP error: len(marshalledSP) < 1")
	}
}

// a subroutine of getUrls above
func (a *ArchonDHT) getUrlFromDHT(ctx context.Context, spProf *RegisteredSp, cacheProfileExists bool, nid, spDataFilepath string) (string, error) {
	nodeUploadKey := "/archonurl/" + nid
	value, err := a.dHT.GetValue(ctx, nodeUploadKey)
	if err == nil {
		url := new(UrlsStruct)
		json_err := json.Unmarshal(value, &url)
		if json_err == nil {
			if cacheProfileExists {
				// put in cache
				updatedProf := spProf

				updatedProf.Url = url.Urls
				marshalledSP, err := json.Marshal(updatedProf)
				if err != nil {
					common.LogError.Println("Caching archonSP error: ", err)
				}
				if len(marshalledSP) > 0 {
					ioutil.WriteFile(spDataFilepath, marshalledSP, 0644) // file is locked in getUrls
				} else {
					common.LogError.Println("Caching archonSP error: len(marshalledSP) < 1")
				}
			}
			return url.Urls, nil
		} else {
			return "", json_err
		}
	}
	return "", err
}

// each key is some pre-image of the cid mapping. depending on implementation, this can be
// an archonNamedUrl + <shardIdx>, archonHashUrl + <shardIdx>, etc
func (a *ArchonDHT) GetUrlsOfNodesHoldingKeys(keys []string, timeoutInSeconds time.Duration) (UrlArray, error) {
	timeoutMessage := make(chan bool, 1)
	go func(s time.Duration) {
		time.Sleep(s)
		timeoutMessage <- true
	}(timeoutInSeconds)

	functionCompleteMessage := make(chan bool, 1)
	retMessage := make(chan UrlArray, 1)
	errMessage := make(chan error, 1)

	go func() {
		keysAsCids, err := StringsToCids(keys)
		if err != nil {
			retMessage <- nil
			errMessage <- err
			functionCompleteMessage <- true
		}
		var wg sync.WaitGroup
		wg.Add(len(keysAsCids))
		providersMessage := make(chan string, len(keysAsCids))
		for i := 0; i < len(keysAsCids); i++ {
			go func(k cid.Cid, w *sync.WaitGroup) {
				defer w.Done()
				key := "/archondl/" + k.String()
				ctx, _ := context.WithTimeout(context.Background(), timeoutInSeconds)
				providers, _ := a.dHT.GetValue(ctx, key)
				downloadUrls := new(UrlsStruct)
				json_err := json.Unmarshal(providers, &downloadUrls)
				if json_err != nil {
					providersMessage <- ""
				}
				providersMessage <- downloadUrls.Urls
			}(keysAsCids[i], &wg)
		}
		wg.Wait()
		var retProvidersString UrlArray
		for i := 0; i < len(keysAsCids); i++ {
			p := <-providersMessage
			if p != "" {
				retProvidersString = append(retProvidersString, p)
			}
		}
		if len(retProvidersString) == 0 {
			retMessage <- nil
			errMessage <- fmt.Errorf("GetUrlsOfNodesHoldingKeys returns 0 urls")
			functionCompleteMessage <- true
		}
		retMessage <- retProvidersString
		errMessage <- nil
		functionCompleteMessage <- true
	}()
	select {
	case <-timeoutMessage:
		return nil, fmt.Errorf("error GetUrlsOfNodesHoldingKeys, timeout")
	case <-functionCompleteMessage:
		return <-retMessage, <-errMessage
	}
}

func (a *ArchonDHT) FindPeer(peerId peer.ID) (peer.AddrInfo, error) {
	return a.dHT.FindPeer(context.Background(), peerId)
}

func (a *ArchonDHT) Connect(peer peer.AddrInfo) error {
	return a.routedHost.Connect(context.Background(), peer)
}

func (a *ArchonDHT) Peers() peer.IDSlice {
	var p peer.IDSlice
	if a.routedHost != nil {
		ps := a.routedHost.Peerstore()
		if ps != nil {
			p = ps.Peers()
		}
	}
	return p
}
