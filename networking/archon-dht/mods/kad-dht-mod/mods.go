package dht

func (dht *IpfsDHT) HasPeers() bool {
	peers := dht.host.Network().Peers()
	if len(peers) > 0 {
		return true
	} else {
		return false
	}
}
