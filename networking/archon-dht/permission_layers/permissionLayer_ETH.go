package permission_layers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/client_utils"
	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/register"
	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/rpc_utils"
	. "github.com/itsmeknt/archoncloud-go/blockchainAPI/registered_sp"
	"github.com/itsmeknt/archoncloud-go/common"
	permLayer "github.com/itsmeknt/archoncloud-go/networking/archon-dht/dht_permission_layer"
)

// ETHEREUM

var eth_blockTime time.Duration = time.Duration(30) // minutes
// FIXME after beta this needs to be the actual eth blocktime otherwise
// there is a gaping security vulnerability to the archon network
// contact george@archon.cloud for details

const cacheFilepath_ETH string = ".sp_profiles_cache/" + string(permLayer.EthPermissionId) + "/"

func BCAddressToEthAddress(address common.BCAddress) [20]byte {
	var bAddress []byte
	sAddress := string(address)
	for i := 0; i < len(address); i += 2 {
		r, _ := hexutil.Decode("0x" + sAddress[i:i+2])
		bAddress = append(bAddress, []byte(r)...)
	}
	var b20Address [20]byte
	for i := 0; i < 20; i++ {
		b20Address[i] = bAddress[i]
	}
	return b20Address
}

func EthAddressToBCAddress(address [20]byte) common.BCAddress {
	var bAddress []byte
	bAddress = make([]byte, 20)
	copy(bAddress[0:len(address)], address[0:len(address)])
	hexAddress := hexutil.Encode(bAddress)
	ret := common.BCAddress(hexAddress)
	return ret
}

var validationInProgress_ETH = struct {
	sync.RWMutex
	m map[peer.ID]bool
}{m: make(map[peer.ID]bool)}

var peerID2ETHAddrs = struct {
	sync.RWMutex
	m map[peer.ID]common.BCAddress
}{m: make(map[peer.ID]common.BCAddress)}

var inPeerstoreMap_ETH = struct {
	sync.RWMutex
	m map[peer.ID]bool
}{m: make(map[peer.ID]bool)}

var checkedButInvalid_ETH = struct {
	sync.RWMutex
	m map[peer.ID]bool
}{m: make(map[peer.ID]bool)}

type Ethereum struct {
}

func (e Ethereum) ID() permLayer.PermissionLayerID {
	return permLayer.EthPermissionId
}

func (e Ethereum) Permissioned() bool {
	return true
}

func (e Ethereum) ValidatePeersPtrArr(bootstrapPeers []*peer.AddrInfo, timeout time.Duration) (res []*peer.AddrInfo, err error) {
	var pBootstrapPeers []peer.AddrInfo
	for _, p := range bootstrapPeers {
		pBootstrapPeers = append(pBootstrapPeers, *p)
	}
	pValidatedPeers, err := e.ValidatePeers(pBootstrapPeers, timeout)
	var validatedPeers []*peer.AddrInfo
	for _, v := range pValidatedPeers {
		if v.ID != "" {
			mV := v
			validatedPeers = append(validatedPeers, &mV)
		}
	}
	return validatedPeers, err
}

func (e Ethereum) inPeerstore(pid peer.ID) bool {
	inPeerstoreMap_ETH.Lock()
	b := inPeerstoreMap_ETH.m[pid]
	inPeerstoreMap_ETH.Unlock()
	return b
}

func (e Ethereum) inCheckedButInvalid(pid peer.ID) bool {
	checkedButInvalid_ETH.Lock()
	b := checkedButInvalid_ETH.m[pid]
	checkedButInvalid_ETH.Unlock()
	return b
}

func (e Ethereum) IsInPeerstoreMap(tocheckPeers []peer.AddrInfo) (isIn, notIn []peer.AddrInfo) {
	var iisIn []peer.AddrInfo
	var nnotIn []peer.AddrInfo
	for i := 0; i < len(tocheckPeers); i++ {
		if e.inPeerstore(tocheckPeers[i].ID) {
			iisIn = append(iisIn, tocheckPeers[i])
		} else {
			if !e.inCheckedButInvalid(tocheckPeers[i].ID) {
				nnotIn = append(nnotIn, tocheckPeers[i])
			}
		}
	}
	return iisIn, nnotIn
}

func (e Ethereum) PutInPeerstoreMap(validated, notValidated []peer.AddrInfo) {
	inPeerstoreMap_ETH.Lock()
	if inPeerstoreMap_ETH.m == nil {
		inPeerstoreMap_ETH.m = make(map[peer.ID]bool)
	}
	inPeerstoreMap_ETH.Unlock()
	checkedButInvalid_ETH.Lock()
	if checkedButInvalid_ETH.m == nil {
		checkedButInvalid_ETH.m = make(map[peer.ID]bool)
	}
	checkedButInvalid_ETH.Unlock()
	for i := 0; i < len(validated); i++ {
		if len(validated[i].ID) > 0 {
			inPeerstoreMap_ETH.Lock()
			inPeerstoreMap_ETH.m[validated[i].ID] = true
			inPeerstoreMap_ETH.Unlock()
		}
	}
	for i := 0; i < len(notValidated); i++ {
		if len(notValidated[i].ID) > 0 {
			checkedButInvalid_ETH.Lock()
			checkedButInvalid_ETH.m[notValidated[i].ID] = true
			checkedButInvalid_ETH.Unlock()
		}
	}
	return
}

func (e Ethereum) ValidatePeer(pid peer.ID, timeout time.Duration) (bool, error) {
	//  TODO USE TIMEOUT
	return ValidatePeerIDETH(pid)
}

func (e Ethereum) ValidatePeers(bootstrapPeers []peer.AddrInfo, timeout time.Duration) (res []peer.AddrInfo, err error) {
	inPeerstore, notInPeerstore := e.IsInPeerstoreMap(bootstrapPeers)
	validated, notValidated, err := ValidateBootstrapPeersETH(notInPeerstore, timeout)
	e.PutInPeerstoreMap(validated, notValidated)
	for i := 0; i < len(inPeerstore); i++ {
		validated = append(validated, inPeerstore[i])
	}
	if err.Error() == "" {
		err = nil
	}
	return validated, err
}

func (e Ethereum) UpdateIndividualSPProfileCache(pid peer.ID) {
	// check if in cache
	filepath := filepath.FromSlash(cacheFilepath_ETH + pid.Pretty())
	spDataFilepath := common.DefaultToExecutable(filepath)
	info, err := os.Stat(spDataFilepath)
	if err != nil {
		if !os.IsNotExist(err) {
			if time.Since(info.ModTime()) < eth_blockTime*time.Minute && info.Size() > 0 {

				return
			}
		}
	} else {
		if time.Since(info.ModTime()) < eth_blockTime*time.Minute && info.Size() > 0 {
			return
		}
	}

	// not in cache
	// check if have ethAddress
	peerID2ETHAddrs.Lock()
	var spAddress [20]byte
	ethAddress, ok := peerID2ETHAddrs.m[pid]
	if ok {
		spAddress = BCAddressToEthAddress(ethAddress)
	} else {
		// if dont have eth address
		bNodeID := []byte(pid.Pretty())
		var b32NodeID [32]byte
		copy(b32NodeID[:], bNodeID[2:])
		ethAddress, err := client_utils.GetNodeID2Address(b32NodeID)
		if err != nil {
			return
		}
		var isAllZeros bool = true
		for i := 0; i < len(ethAddress); i++ {
			if ethAddress[i] != byte(0) {
				isAllZeros = false
			}
		}
		if isAllZeros {
			return
		}
		inGoodStanding, err := register.CheckIfInGoodStanding_byteAddress(ethAddress)
		if err != nil {
			return
		}
		if !inGoodStanding {
			return
		}
		if peerID2ETHAddrs.m == nil {
			peerID2ETHAddrs.m = make(map[peer.ID]common.BCAddress)
		}
		peerID2ETHAddrs.m[pid] = EthAddressToBCAddress(ethAddress)
		spAddress = ethAddress
	}
	peerID2ETHAddrs.Unlock()
	// get spprofile
	sp, err := client_utils.GetRegisteredSP(spAddress)
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

func (e Ethereum) UpdateSPProfileCache(pids []peer.ID) {
	//common.LogDebug.Println("Ethereum UpdateSPProfileCache() pids ", pids)
	go func() {
		err := os.MkdirAll(path.Dir(cacheFilepath_ETH), os.ModeDir|os.ModePerm)
		if err != nil {
			common.LogError.Println("UpdateSPProfileCache() error: ", err)
		}
		for i := 0; i < len(pids); i++ {
			go func(pid peer.ID, ee Ethereum) {
				ee.UpdateIndividualSPProfileCache(pid)
			}(pids[i], e)
		}
	}()
	//common.LogDebug.Println("Ethereum UpdateSPProfileCache() returns")
	return
}

// TODO PUT ELSEWHERE

func CompareHexOrDec(lhs, rhs string) (int, error) {
	// -1 for lhs
	// 1 for rhs
	lhsRune := []rune(lhs)
	rhsRune := []rune(rhs)
	if len(lhsRune) > len(rhsRune) {
		return -1, nil
	} else if len(lhsRune) < len(rhsRune) {
		return 1, nil
	}
	for i := len(lhsRune) - 1; i > 0; i-- {
		parsedDigLHS, err := strconv.ParseUint(string(lhsRune[i]), 16, 64)
		if err != nil {
			return 0, err
		}
		parsedDigRHS, err := strconv.ParseUint(string(rhsRune[i]), 16, 64)
		if err != nil {
			return 0, err
		}
		if parsedDigLHS > parsedDigRHS {
			return -1, nil
		} else if parsedDigLHS > parsedDigRHS {
			return 1, nil
		}
		// next digit
	}
	return 0, nil // they are equal

}

func (e Ethereum) CompareBlockHeights(lhs, rhs permLayer.VersionData) (int, error) {
	// returns -1 for lhs
	// returns 1 for rhs
	// returns 0 for undecided (error)
	// this is called by select

	// compare blockHeight
	// note: different clients use different bases, so comparison will be naive
	res, err := CompareHexOrDec(lhs.BlockHeight[2:], rhs.BlockHeight[2:])
	if err != nil {
		return 0, err
	}
	// for latest, check that corresponding blockHash is correct
	candidate := *new(permLayer.VersionData)
	runnerUp := *new(permLayer.VersionData)
	var ret int
	if res <= 0 {
		candidate = lhs
		ret = -1
		runnerUp = rhs
	} else {
		candidate = rhs
		ret = 1
		runnerUp = lhs
	}

	blockHash, err := rpc_utils.GetBlockHash(candidate.BlockHeight)
	if err == nil {
		if candidate.BlockHash == blockHash {
			return ret, nil
		}
	}
	ret = -ret

	// if blockHash does not match candidate.BlockHash, do same (check blockHash) for other
	blockHash, err = rpc_utils.GetBlockHash(runnerUp.BlockHeight)
	if err == nil {
		if runnerUp.BlockHash == blockHash {
			return ret, nil
		}
	}
	return 0, fmt.Errorf("error CompareBlockHeights: comparison failed")
}

func (e Ethereum) GetBlockHeight() (string, error) {
	return rpc_utils.GetBlockHeight()
}

func (e Ethereum) GetBlockHash(blockHeight string) (string, error) {
	return rpc_utils.GetBlockHash(blockHeight)
}

func (e Ethereum) NewVersionData() (*permLayer.VersionData, error) {
	blockHeight, err := e.GetBlockHeight()
	if err != nil {
		return nil, err
	}
	blockHash, err := e.GetBlockHash(blockHeight)
	if err != nil {
		return nil, err
	}
	versionData := new(permLayer.VersionData)
	versionData.BlockHeight = blockHeight
	versionData.BlockHash = blockHash
	return versionData, nil
}

// END ETHEREUM
