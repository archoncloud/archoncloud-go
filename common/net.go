package common

import (
	"fmt"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"net"
)

func GetMultiAddressOf(ipOrDns string) (ma.Multiaddr, error) {
	ips, err := net.LookupIP(ipOrDns)
	if err != nil {return nil, err}

	for _, ip := range ips {
		a, err := manet.FromIP(ip)
		if err == nil {
			if !manet.IsPrivateAddr(a) {
				return a, nil
			}
		}
	}
	return nil, fmt.Errorf("no multi-addr for %s", ipOrDns)
}
