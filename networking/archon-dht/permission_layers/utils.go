package permission_layers

import (
	"strings"

	permLayer "github.com/itsmeknt/archoncloud-go/networking/archon-dht/dht_permission_layer"
)

func NewPermissionLayer(sid string) permLayer.PermissionLayer {
	id := permLayer.PermissionLayerID(strings.ToUpper(sid))
	switch id {
	case permLayer.EthPermissionId:
		return Ethereum{}
	case permLayer.NeoPermissionId:
		return Neo{}
	case permLayer.NotPermissionId:
		return NonPermissioned{}
	default:
		return nil
	}
}

type VersionData permLayer.VersionData

var SpFilenames = permLayer.SpFilenames
