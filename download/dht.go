package download

import (
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"time"
)

//	GetDownloadUrls returns a map from shard index to urls storing it
func GetDownloadUrls(aUrl *ArchonUrl) (map[int][]string, error) {
	for _, s := range GetAllSeedUrls() {
		fmt.Printf("Querying seed %s...\n", s)
		contents, err := GetFromSP(s, RetrieveEndpoint, "url="+aUrl.String(), 15*time.Second)
		if err == nil {
			r, err := NewRetrieveResponse([]byte(contents));
			if err == nil {
				return r.Urls, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find SPs storing %s", aUrl)
}
