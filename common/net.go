package common

import (
	"fmt"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"net"
	"net/url"
	"strconv"
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

// If port is non-zero it will be used for all the urls
func FirstLiveUrl(urls []string, port int) string {
	for _, urlS := range urls {
		url, err := url.Parse(urlS)
		if err != nil {continue}
		network := url.Host
		if port != 0 {
			p := strings.Split(network,":")
			network = p[0] + ":" + strconv.Itoa(port)
		}
		err = tryPort(network, 3*time.Second)
		if err == nil {
			if url.Scheme != "" {
				return url.Scheme + "://" + network
			}
			return network
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