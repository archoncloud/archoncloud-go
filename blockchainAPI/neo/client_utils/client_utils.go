package client_utils

import (
	"github.com/archoncloud/archoncloud-go/blockchainAPI/neo"
	. "github.com/archoncloud/archoncloud-go/blockchainAPI/registered_sp"
	"github.com/pariz/gountries"
)

func GetNodeID2Address(nodeID string) (addr string, err error) {
	addrS, err := neo.GetSpAddress(string(nodeID))
	if err != nil {
		return
	}
	addr = addrS
	return
}

func GetRegisteredSP(neoAddress string) (sp *RegisteredSp, err error) {
	prof, err := neo.GetSpProfile(string(neoAddress))
	if err != nil {
		return
	}

	sp = new(RegisteredSp)
	sp.Address = neoAddress
	sp.PledgedStorage = uint64(prof.PledgedStorage)
	sp.CountryCode = gountries.Codes{Alpha3: prof.CountryA3}
	sp.MinAskPrice = uint64(prof.MinAsk)
	sp.NodeID = prof.NodeId
	return
}
