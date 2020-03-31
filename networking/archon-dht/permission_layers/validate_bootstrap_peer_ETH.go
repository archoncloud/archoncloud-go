package permission_layers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/client_utils"
	"github.com/itsmeknt/archoncloud-go/blockchainAPI/ethereum/register"
	"github.com/itsmeknt/archoncloud-go/common"

	"github.com/libp2p/go-libp2p-core/peer"
)

// ETH
func ValidatePeerIDETH(bootstrapPeer peer.ID) (ret bool, err error) {
	//fmt.Println("debug validation pid ", bootstrapPeer)
	var dummy bool = false
	validationInProgress_ETH.Lock()
	if validationInProgress_ETH.m == nil {
		validationInProgress_ETH.m = make(map[peer.ID]bool)
	}
	validationInProgress_ETH.m[bootstrapPeer] = true
	validationInProgress_ETH.Unlock()
	nodeID := bootstrapPeer.Pretty()
	/*fmt.Println("debug nodeID ", nodeID)
	if nodeID == "QmVwAp1BQHXxtzrCqbvzYLHFxmm7RaaGjMfSfsMbESjXJe" { // TEST
	*/ /*if nodeID == "QmQ9bD1cXDNE8iRU7sgVBAkCyLuSw5cPacnMDvDKAgiCPE" {
		fmt.Println("DEBUG TEST")
		return dummy, nil
	}*/ // was useful test
	bNodeID := []byte(nodeID)
	var b32NodeID [32]byte
	copy(b32NodeID[:], bNodeID[2:])
	ethAddress, err := client_utils.GetNodeID2Address(b32NodeID)
	if err != nil {
		return dummy, nil
	}
	var isAllZeros bool = true
	for i := 0; i < len(ethAddress); i++ {
		if ethAddress[i] != byte(0) {
			isAllZeros = false
		}
	}
	if isAllZeros {
		return dummy, nil
	}
	inGoodStanding, err := register.CheckIfInGoodStanding_byteAddress(ethAddress)
	if err != nil {
		return dummy, nil
	}
	if !inGoodStanding {
		return false, nil
	}
	peerID2ETHAddrs.Lock()
	if peerID2ETHAddrs.m == nil {
		peerID2ETHAddrs.m = make(map[peer.ID]common.BCAddress)
	}
	peerID2ETHAddrs.m[bootstrapPeer] = EthAddressToBCAddress(ethAddress)
	peerID2ETHAddrs.Unlock()
	validationInProgress_ETH.Lock()
	if validationInProgress_ETH.m == nil {
		validationInProgress_ETH.m = make(map[peer.ID]bool)
	}
	validationInProgress_ETH.m[bootstrapPeer] = false
	validationInProgress_ETH.Unlock()
	return true, nil
}

func ValidateBootstrapPeersETH(bootstrapPeers []peer.AddrInfo, timeout time.Duration) (val, notval []peer.AddrInfo, err error) {

	var retVal, retNotVal []peer.AddrInfo
	if len(bootstrapPeers) == 0 {
		return retVal, retNotVal, fmt.Errorf("error ValidateBootstrapPeers: bootstrapPeers array empty")
	}
	validationInProgress_ETH.Lock()
	if validationInProgress_ETH.m == nil {
		validationInProgress_ETH.m = make(map[peer.ID]bool)
	}
	validationInProgress_ETH.Unlock()
	timeoutMessage := make(chan bool, 1)
	go func(s time.Duration) {
		time.Sleep(s)
		timeoutMessage <- true
	}(timeout)
	validatedBootstrapPeersMessage := make(chan peer.AddrInfo, len(bootstrapPeers))
	notValidatedBootstrapPeersMessage := make(chan peer.AddrInfo, len(bootstrapPeers))
	validatedBootstrapPeersMessage_err := make(chan error, len(bootstrapPeers))
	allHaveArrivedMessage := make(chan bool, 1)
	//
	go func(bsPeers []peer.AddrInfo) {
		var wg sync.WaitGroup
		wg.Add(len(bsPeers))
		bsCompleteMessage := make(chan bool, len(bsPeers))
		for i := 0; i < len(bsPeers); i++ {
			bsPeer := bsPeers[i]
			validationInProgress_ETH.Lock()
			if validationInProgress_ETH.m[bsPeer.ID] {
				validationInProgress_ETH.Unlock()
				continue
			}
			validationInProgress_ETH.m[bsPeer.ID] = true
			validationInProgress_ETH.Unlock()
			go func(bootstrapPeer peer.AddrInfo, wg *sync.WaitGroup) {
				defer wg.Done()
				validated, err := ValidatePeerIDETH(bootstrapPeer.ID)
				if err != nil {
					validatedBootstrapPeersMessage_err <- err
					validatedBootstrapPeersMessage <- *new(peer.AddrInfo)
					bsCompleteMessage <- true
					return
				}

				if validated {
					notValidatedBootstrapPeersMessage <- *new(peer.AddrInfo)
					validatedBootstrapPeersMessage <- bootstrapPeer
					validatedBootstrapPeersMessage_err <- nil
				} else {
					notValidatedBootstrapPeersMessage <- bootstrapPeer
					validatedBootstrapPeersMessage <- *new(peer.AddrInfo)
					validatedBootstrapPeersMessage_err <- fmt.Errorf("error ValidateBootstrapPeers, peer is not registered w permission layer")
				}
				bsCompleteMessage <- true
			}(bsPeer, &wg)
		}
		wg.Wait()
		allHaveArrivedMessage <- true
	}(bootstrapPeers)
	// ALL HAVE ARRIVED
	var retErr error
	select {
	case <-timeoutMessage:
		for i := 0; i < len(bootstrapPeers); i++ {
			retVal = append(retVal, <-validatedBootstrapPeersMessage)
			retNotVal = append(retNotVal, <-notValidatedBootstrapPeersMessage)
		}
		var errArray []string
		for i := 0; i < len(bootstrapPeers); i++ {
			e := <-validatedBootstrapPeersMessage_err
			if e != nil {
				errArray = append(errArray, e.Error())
			}
		}
		retErr = fmt.Errorf(strings.Join(errArray, "\n"))
		return retVal, retNotVal, retErr
	case <-allHaveArrivedMessage:
		for i := 0; i < len(bootstrapPeers); i++ {
			retVal = append(retVal, <-validatedBootstrapPeersMessage)
			retNotVal = append(retNotVal, <-notValidatedBootstrapPeersMessage)
		}
		var errArray []string
		for i := 0; i < len(bootstrapPeers); i++ {
			e := <-validatedBootstrapPeersMessage_err
			if e != nil {
				errArray = append(errArray, e.Error())
			}
		}
		retErr = fmt.Errorf(strings.Join(errArray, "\n"))
		return retVal, retNotVal, retErr
	}
}
