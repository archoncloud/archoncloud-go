package archon_dht

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/itsmeknt/archoncloud-go/networking/archon-dht/dht_permission_layer"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/itsmeknt/archoncloud-go/common"

	"github.com/libp2p/go-libp2p-core/peer"
	mh "github.com/multiformats/go-multihash"
)

func (a ArchonDHTs) pollUpdateSPProfileCache(interval time.Duration) {
	go func(i time.Duration, arc ArchonDHTs) {
		for {
			arc.updateSPProfileCaches()
			time.Sleep(i)
		}
	}(interval, a)
}

func (a ArchonDHTs) updateSPProfileCaches() {
	for k, v := range a.Layers {
		if k != "NON" {
			v.updateSPProfileCache()
		}
	}
}

func (a ArchonDHTs) pollAnnounceUrl(interval time.Duration) {
	go func(i time.Duration, arc ArchonDHTs) {
		for {
			arc.announceUrl()
			time.Sleep(i)
		}
	}(interval, a)
}

func (a ArchonDHTs) announceUrl() {
	var nodeID string
	foundAnID := false
	for foundAnID == false {
		for _, v := range a.Layers {
			if v.routedHost != nil {
				nodeID = string(v.routedHost.ID())
				foundAnID = true
				break // we only want one
			}
		}
	}
	keyhash := []byte(nodeID)
	nodeIDAsMh, err := mh.Cast(keyhash)
	if err != nil {
		common.LogError.Println(err)
	}
	nodeIDAsCid := cid.NewCidV0(nodeIDAsMh)
	time.Sleep(4 * time.Second)
	// delay in case network just booted
	_ = a.putValue(nodeIDAsCid, "/archonurl/")
}

func (a *ArchonDHTs) putValue(keyAsCid cid.Cid, archonPrefix string) error { // archonPrefix is /archonurl/
	var wg sync.WaitGroup
	wg.Add(len(a.Layers))
	errMessage := make(chan error, len(a.Layers))
	for _, v := range a.Layers {
		go func(vDht *ArchonDHT, wwg *sync.WaitGroup) {
			defer wwg.Done()
			// wait until bootstrapped
			for {
				if vDht.dHT != nil {
					if vDht.dHT.HasPeers() {
						break
					} else {
						select {
						case <-time.After(200 * time.Millisecond):
							continue
						}
					}
				} else {
					select {
					case <-time.After(200 * time.Millisecond):
						continue
					}
				}
			}
			var err error
			if archonPrefix == "/archonurl/" {
				err = vDht.putUrl(archonPrefix, keyAsCid)
			}
			errMessage <- err
		}(v, &wg)
	}
	wg.Wait()
	var errString string
	for i := 0; i < len(a.Layers); i++ {
		e := <-errMessage
		if e != nil {
			errString += e.Error()
		}
	}
	if len(errString) > 0 {
		return fmt.Errorf(errString)
	}
	return nil
}

func (a *ArchonDHTs) putValueVersioned(archonPrefix string, keyAsCid cid.Cid, versionData dht_permission_layer.VersionData) error { // archonPrefix is /archondl/
	var wg sync.WaitGroup
	wg.Add(len(a.Layers))
	errMessage := make(chan error, len(a.Layers))
	for _, v := range a.Layers {
		go func(vDht *ArchonDHT, wwg *sync.WaitGroup) {
			defer wwg.Done()
			// wait until bootstrapped
			for {
				if vDht.dHT != nil {
					if vDht.dHT.HasPeers() {
						break
					} else {
						select {
						case <-time.After(200 * time.Millisecond):
							continue
						}
					}
				} else {
					select {
					case <-time.After(200 * time.Millisecond):
						continue
					}
				}
			}
			var err error
			if archonPrefix == "/archondl/" {
				err = vDht.putUrlVersioned(archonPrefix, keyAsCid, versionData)
			}
			errMessage <- err
		}(v, &wg)
	}
	wg.Wait()
	var errString string
	for i := 0; i < len(a.Layers); i++ {
		e := <-errMessage
		if e != nil {
			errString += e.Error()
		}
	}
	if len(errString) > 0 {
		return fmt.Errorf(errString)
	}
	return nil
}

type bundled struct {
	PeerAddrInfo UrlArray
	Error        error
	Key          dht_permission_layer.PermissionLayerID
}

///////////////////////////////////////////////////////////////////////////
/// below are on single layer /////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////
/// this can be seen as the difference between ArchonDHTs and ArchonDHT ///
///////////////////////////////////////////////////////////////////////////

func (a *ArchonDHT) updateSPProfileCache() {
	p := a.Peers()
	var connectedPeers []peer.ID
	for i := 0; i < len(p); i++ {
		connected := a.routedHost.Network().Connectedness(p[i])
		if connected == 1 {
			connectedPeers = append(connectedPeers, p[i])
		}
	}
	connectedPeers = append(connectedPeers, a.routedHost.ID()) // self
	a.Config.PermissionLayer.UpdateSPProfileCache(connectedPeers)
	a.updateUrls(connectedPeers)
}

func (a *ArchonDHT) updateUrls(connectedPeers []peer.ID) {
	go func(ps []peer.ID) {
		sps := make([]string, len(ps))
		for i := 0; i < len(ps); i++ {
			sps = append(sps, ps[i].Pretty())
		}
		to := 3 * time.Second
		_, _ = a.getUrls(sps, to)
	}(connectedPeers)
}

// called by putValue
func (d *ArchonDHT) putUrl(archonPrefix string, keyAsCid cid.Cid) error {
	var archonUlKey string = archonPrefix + keyAsCid.String()
	var ULUs UrlsStruct = UrlsStruct{Urls: d.Config.Url}
	uploadUrls, err := json.Marshal(ULUs)
	if err != nil {
		return err
	}
	return d.dHT.PutValue(context.Background(), archonUlKey, uploadUrls)
}

func (d *ArchonDHT) putUrlVersioned(archonPrefix string, keyAsCid cid.Cid, versionData dht_permission_layer.VersionData) error {
	var archonUlKey string = archonPrefix + keyAsCid.String()
	var ULUs UrlsVersionedStruct = UrlsVersionedStruct{Urls: d.Config.Url, Versioning: versionData}
	downloadUrlsVersioned, err := json.Marshal(ULUs)
	if err != nil {
		return err
	}
	return d.dHT.PutValue(context.Background(), archonUlKey, downloadUrlsVersioned)
}
