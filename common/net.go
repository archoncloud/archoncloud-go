package common

import (
	"fmt"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"net"
	"strings"
	"time"
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

func FirstLiveUrl(urls []string) string {
	for _, url := range urls {
		// Note assuming http
		err := tryPort(strings.TrimPrefix(url,"http://"), 3*time.Second)
		if err == nil {
			return url
		}
	}
	return ""
}

func tryPort(network string, timeout time.Duration) (err error) {
	conn, err := net.DialTimeout("tcp", network, timeout)
	if conn != nil {
		conn.Close()
	}
	return
}