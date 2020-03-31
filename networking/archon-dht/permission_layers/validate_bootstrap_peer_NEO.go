package permission_layers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/itsmeknt/archoncloud-go/blockchainAPI/neo/client_utils"
	"github.com/itsmeknt/archoncloud-go/common"
	"github.com/libp2p/go-libp2p-core/peer"
)

// NEO

func ValidatePeerIDNEO(bootstrapPeer peer.ID) (ret bool, err error) {
	var dummy bool = false
	validationInProgress_NEO.Lock()
	if validationInProgress_NEO.m == nil {
		validationInProgress_NEO.m = make(map[peer.ID]bool)
	}
	validationInProgress_NEO.m[bootstrapPeer] = true
	validationInProgress_NEO.Unlock()
	/*fmt.Println("debug nodeID ", nodeID)
	if nodeID == "QmVwAp1BQHXxtzrCqbvzYLHFxmm7RaaGjMfSfsMbESjXJe" { // TEST
		fmt.Println("DEBUG TEST")
		return dummy, nil
	}*/ // was useful test
	// If no error is returned, SP is registered
	neoAddress, err := client_utils.GetNodeID2Address(bootstrapPeer.Pretty())
	if err != nil {
		return dummy, nil
	}
	peerID2NEOAddrs.Lock()
	if peerID2NEOAddrs.m == nil {
		peerID2NEOAddrs.m = make(map[peer.ID]common.BCAddress)
	}
	peerID2NEOAddrs.m[bootstrapPeer] = neoAddress
	peerID2NEOAddrs.Unlock()
	validationInProgress_NEO.Lock()
	if validationInProgress_NEO.m == nil {
		validationInProgress_NEO.m = make(map[peer.ID]bool)
	}
	validationInProgress_NEO.m[bootstrapPeer] = false
	validationInProgress_NEO.Unlock()
	return true, nil
}

// TODO THIS FUNCTION CAN BE MADE GENERIC but for now it should work

func ValidateBootstrapPeersNEO(bootstrapPeers []peer.AddrInfo, timeout time.Duration) (val, notval []peer.AddrInfo, err error) {

	var retVal, retNotVal []peer.AddrInfo
	if len(bootstrapPeers) == 0 {
		return retVal, retNotVal, fmt.Errorf("error ValidateBootstrapPeers: bootstrapPeers array empty")
	}
	validationInProgress_NEO.Lock()
	if validationInProgress_NEO.m == nil {
		validationInProgress_NEO.m = make(map[peer.ID]bool)
	}
	validationInProgress_NEO.Unlock()
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
			validationInProgress_NEO.Lock()
			if validationInProgress_NEO.m[bsPeer.ID] {
				validationInProgress_NEO.Unlock()
				continue
			}
			validationInProgress_NEO.m[bsPeer.ID] = true
			validationInProgress_NEO.Unlock()
			go func(bootstrapPeer peer.AddrInfo, wg *sync.WaitGroup) {
				defer wg.Done()
				validated, err := ValidatePeerIDNEO(bootstrapPeer.ID)
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
	return retVal, retNotVal, nil
}
