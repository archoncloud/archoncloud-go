package permission_layers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/itsmeknt/archoncloud-go/blockchainAPI/neo"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/itsmeknt/archoncloud-go/blockchainAPI/neo/client_utils"
	. "github.com/itsmeknt/archoncloud-go/blockchainAPI/registered_sp"
	"github.com/itsmeknt/archoncloud-go/common"
	permLayer "github.com/itsmeknt/archoncloud-go/networking/archon-dht/dht_permission_layer"
)

// NEO

var neo_blockTime time.Duration = time.Duration(30) // minutes
// FIXME after beta this needs to be the actual neo blocktime otherwise
// there is a gaping security vulnerability to the archon network
// contact george@archon.cloud for details

const cacheFilepath_NEO string = ".sp_profiles_cache/" + string(permLayer.NeoPermissionId) + "/"

func BCAddressToNEOAddress(address common.BCAddress) string {
	return string(address)
}

func NEOAddressToBCAddress(address string) common.BCAddress {
	return common.BCAddress(address)
}

var validationInProgress_NEO = struct {
	sync.RWMutex
	m map[peer.ID]bool
}{m: make(map[peer.ID]bool)}

var peerID2NEOAddrs = struct {
	sync.RWMutex
	m map[peer.ID]common.BCAddress
}{m: make(map[peer.ID]common.BCAddress)}

var inPeerstoreMap_NEO = struct {
	sync.RWMutex
	m map[peer.ID]bool
}{m: make(map[peer.ID]bool)}

var checkedButInvalid_NEO = struct {
	sync.RWMutex
	m map[peer.ID]bool
}{m: make(map[peer.ID]bool)}

type Neo struct {
}

func (n Neo) ID() permLayer.PermissionLayerID {
	return permLayer.NeoPermissionId
}

func (n Neo) Permissioned() bool {
	return true
}

func (n Neo) ValidatePeersPtrArr(bootstrapPeers []*peer.AddrInfo, timeout time.Duration) (res []*peer.AddrInfo, err error) {
	var pBootstrapPeers []peer.AddrInfo
	for _, p := range bootstrapPeers {
		pBootstrapPeers = append(pBootstrapPeers, *p)
	}
	pValidatedPeers, err := n.ValidatePeers(pBootstrapPeers, timeout)
	var validatedPeers []*peer.AddrInfo
	for _, v := range pValidatedPeers {
		if v.ID != "" {
			mV := v
			validatedPeers = append(validatedPeers, &mV)
		}
	}
	return validatedPeers, err
}

// TODO REFACTOR PEERSTORE MAP FUNCTIONS TO BE GENERIC ACROSS ETH,NEO ETC
// for now this should work

func (n Neo) inPeerstore(pid peer.ID) bool {
	inPeerstoreMap_NEO.Lock()
	b := inPeerstoreMap_NEO.m[pid]
	inPeerstoreMap_NEO.Unlock()
	return b
}

func (n Neo) inCheckedButInvalid(pid peer.ID) bool {
	checkedButInvalid_NEO.Lock()
	b := checkedButInvalid_NEO.m[pid]
	checkedButInvalid_NEO.Unlock()
	return b
}

func (n Neo) IsInPeerstoreMap(tocheckPeers []peer.AddrInfo) (isIn, notIn []peer.AddrInfo) {
	var iisIn []peer.AddrInfo
	var nnotIn []peer.AddrInfo
	for i := 0; i < len(tocheckPeers); i++ {
		if n.inPeerstore(tocheckPeers[i].ID) {
			iisIn = append(iisIn, tocheckPeers[i])
		} else {
			if !n.inCheckedButInvalid(tocheckPeers[i].ID) {
				nnotIn = append(nnotIn, tocheckPeers[i])
			}
		}
	}
	return iisIn, nnotIn
}

func (n Neo) PutInPeerstoreMap(validated, notValidated []peer.AddrInfo) {
	inPeerstoreMap_NEO.Lock()
	if inPeerstoreMap_NEO.m == nil {
		inPeerstoreMap_NEO.m = make(map[peer.ID]bool)
	}
	inPeerstoreMap_NEO.Unlock()
	checkedButInvalid_NEO.Lock()
	if checkedButInvalid_NEO.m == nil {
		checkedButInvalid_NEO.m = make(map[peer.ID]bool)
	}
	checkedButInvalid_NEO.Unlock()
	for i := 0; i < len(validated); i++ {
		if len(validated[i].ID) > 0 {
			inPeerstoreMap_NEO.Lock()
			inPeerstoreMap_NEO.m[validated[i].ID] = true
			inPeerstoreMap_NEO.Unlock()
		}
	}
	for i := 0; i < len(notValidated); i++ {
		if len(notValidated[i].ID) > 0 {
			checkedButInvalid_NEO.Lock()
			checkedButInvalid_NEO.m[notValidated[i].ID] = true
			checkedButInvalid_NEO.Unlock()
		}
	}
	return
}

func (n Neo) ValidatePeer(pid peer.ID, timeout time.Duration) (bool, error) {
	// TODO USE TIMEOUT
	return ValidatePeerIDNEO(pid)
}

func (n Neo) ValidatePeers(bootstrapPeers []peer.AddrInfo, timeout time.Duration) (res []peer.AddrInfo, err error) {
	inPeerstore, notInPeerstore := n.IsInPeerstoreMap(bootstrapPeers)
	validated, notValidated, err := ValidateBootstrapPeersNEO(notInPeerstore, timeout)
	n.PutInPeerstoreMap(validated, notValidated)
	for i := 0; i < len(inPeerstore); i++ {
		validated = append(validated, inPeerstore[i])
	}
	if err.Error() == "" {
		err = nil
	}
	return validated, err
}

func (n Neo) UpdateIndividualSPProfileCache(pid peer.ID) {
	// check if in cache
	filepath := filepath.FromSlash(cacheFilepath_NEO + pid.Pretty())
	spDataFilepath := common.DefaultToExecutable(filepath)
	info, err := os.Stat(spDataFilepath)
	if err != nil {
		if !os.IsNotExist(err) {
			if time.Since(info.ModTime()) < neo_blockTime*time.Minute && info.Size() > 0 {
				return
			}
		}
	} else {
		if time.Since(info.ModTime()) < neo_blockTime*time.Minute && info.Size() > 0 {
			return
		}
	}

	// not in cache
	// check if have neoAddress
	peerID2NEOAddrs.Lock()
	var spAddress string
	neoAddress, ok := peerID2NEOAddrs.m[pid]
	if ok {
		spAddress = BCAddressToNEOAddress(neoAddress)
	} else {
		// if dont have neo address
		neoAddress, err := client_utils.GetNodeID2Address(pid.Pretty())
		if err != nil {
			return
		}
		var isAllZeros bool = true
		for i := 0; i < len(neoAddress); i++ {
			if neoAddress[i] != byte(0) {
				isAllZeros = false
			}
		}
		if isAllZeros {
			return
		}
		inGoodStanding := neo.IsSpRegistered(BCAddressToNEOAddress(neoAddress))
		if !inGoodStanding {
			return
		}
		if peerID2NEOAddrs.m == nil {
			peerID2NEOAddrs.m = make(map[peer.ID]common.BCAddress)
		}
		peerID2NEOAddrs.m[pid] = neoAddress
		spAddress = BCAddressToNEOAddress(neoAddress)
	}
	peerID2NEOAddrs.Unlock()
	// get spprofile
	sp, err := client_utils.GetRegisteredSP(NEOAddressToBCAddress(spAddress))
	if err != nil {
		common.LogError.Println("UpdateSPProfileCache() error: ", pid, err)
	}
	sp.NodeID = pid.Pretty()
	sp.Url = "" // for compare to old below
	marshalledSP, err := json.Marshal(sp)
	if err != nil {
		common.LogError.Println("UpdateSPProfileCache() error: ", pid, err)
	}
	// Check if sp stats are different
	SpFilenames.Lock()
	if SpFilenames.M[spDataFilepath] == nil {
		SpFilenames.M[spDataFilepath] = new(permLayer.SafeFilename)
	}
	ptr := SpFilenames.M[spDataFilepath]
	SpFilenames.Unlock()
	ptr.Lock()
	defer ptr.Unlock()
	// get old cached sp profile
	oldSPData, err := ioutil.ReadFile(spDataFilepath)
	if err == nil {
		if len(oldSPData) > 0 {
			oldSPProf := new(RegisteredSp)
			json_err := json.Unmarshal(oldSPData, &oldSPProf)
			if json_err != nil {
				common.LogError.Println("UpdateSPProfileCache() error: ", pid, json_err)
			}
			oldSPProf.Url = "" // set this empty like sp
			marshalledOldSP, err := json.Marshal(oldSPProf)
			if err == nil {
				// compare old w new
				if len(marshalledSP) == len(marshalledOldSP) {
					var same bool = true
					for j := 0; j < len(marshalledSP); j++ {
						if marshalledSP[j] != marshalledOldSP[j] {
							same = false
							break
						}
					}
					if same {
						currentTime := time.Now().Local()
						os.Chtimes(spDataFilepath, currentTime, currentTime)
						return
					}
				}
			}
		}
	}
	// STORE IN CACHE THIS SPPROFILE
	if len(marshalledSP) > 0 {

		ioutil.WriteFile(spDataFilepath, marshalledSP, 0644)
	} else {
		common.LogError.Println("UpdateSPProfileCache() error: len(marshalledSP) < 1 ", pid)
	}
}

func (n Neo) UpdateSPProfileCache(pids []peer.ID) {
	go func() {
		err := os.MkdirAll(path.Dir(cacheFilepath_NEO), os.ModeDir|os.ModePerm)
		if err != nil {
			common.LogError.Println("UpdateSPProfileCache() error: ", err)
		}
		for i := 0; i < len(pids); i++ {
			go func(pid peer.ID, nn Neo) {
				nn.UpdateIndividualSPProfileCache(pid)
			}(pids[i], n)
		}
	}()
	//common.LogDebug.Println("Ethereum UpdateSPProfileCache() returns")
	return
}

func (n Neo) CompareBlockHeights(lhs, rhs permLayer.VersionData) (int, error) {
	//  TODO
	return 0, nil
}

func (n Neo) GetBlockHeight() (string, error) {
	return neo.GetBlockHeight()
}

func (n Neo) GetBlockHash(blockHeight string) (string, error) {
	return neo.GetBlockHash(blockHeight)
}

func (n Neo) NewVersionData() (v *permLayer.VersionData, err error) {
	height, err := n.GetBlockHeight()
	if err != nil {
		return
	}
	hash, err := n.GetBlockHash(height)
	if err != nil {
		return
	}
	v = &permLayer.VersionData{
		BlockHeight: height,
		BlockHash:   hash,
	}
	return
}

// END NEO
