package common

import "strconv"

const BootStrapNodeId = "QmNX6ASyukLch38D2Z1h4cMh39ATfqqDom1xJWv2YHc1eG"
const Localhost = "192.168.1.161"	// Debugging only

// The Archon seeds must always provide an http endpoint and participate in all permission layers
var seedUrls = []string { "http://miner1.archon.cloud", "http://miner2.archon.cloud", "http://miner3.archon.cloud",
	"http://miner4.archon.cloud","http://miner5.archon.cloud", "http://miner6.archon.cloud",}

func GetSeedUrls(wanted int) []string {
	if BuildConfig == Debug {
		return []string{"http://localhost"}
	}
	if wanted > len(seedUrls) {
		wanted = len(seedUrls)
	}
	r := RandomIntRange(wanted)
	urls := []string{}
	for _, ix := range r {
		urls = append(urls,seedUrls[ix] + ":" + strconv.Itoa(SeedsPort()))
	}
	return urls
}

func GetAllSeedUrls() []string {
	if BuildConfig == Debug {
		return []string{"http://localhost"}
	}
	urls := []string{}
	for _, url := range seedUrls {
		urls = append(urls, url + ":" + strconv.Itoa(SeedsPort()))
	}
	return urls
}

func SeedsPort() int {
	switch BuildConfig {
	case Debug: return 8000
	case Beta: return 9000
	default: return 0
	}
}