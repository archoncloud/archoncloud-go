package dht_permission_layer

import (
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

type PermissionLayerID string

type VersionData struct {
	BlockHeight string `json:block_height` // height and hash "time"
	BlockHash   string `json:block_hash`   // from upload
}

const (
	EthPermissionId PermissionLayerID = "ETH"
	NotPermissionId PermissionLayerID = "NON"
	NeoPermissionId PermissionLayerID = "NEO"
)

type PermissionLayer interface {
	ID() PermissionLayerID
	Permissioned() bool
	ValidatePeersPtrArr(bootstrapPeers []*peer.AddrInfo, timeout time.Duration) (res []*peer.AddrInfo, err error)
	ValidatePeers(bootstrapPeers []peer.AddrInfo, timeout time.Duration) (res []peer.AddrInfo, err error)
	ValidatePeer(pid peer.ID, timeout time.Duration) (bool, error)
	UpdateIndividualSPProfileCache(pid peer.ID)
	UpdateSPProfileCache(pids []peer.ID)
	CompareBlockHeights(lhs, rhs VersionData) (int, error)
	//GetBlockHeight() (string, error)
	//GetBlockHash(blockHeight string) (string, error)
	NewVersionData() (*VersionData, error)
}
