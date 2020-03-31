package common

import (
	"fmt"
	//permLayer "github.com/archoncloud/archoncloud-go/networking/archon-dht/permission_layer"
	"github.com/pkg/errors"
	"net/url"
	"strconv"
	"strings"
)

// Example: arc://george.eth.n2:7/photos/beach.jpg

type UrlPermission string

const (
	Eth UrlPermission = "eth"
	Neo UrlPermission = "neo"
)

type ArchonUrl struct {
	Username   string
	Permission UrlPermission
	Path       string	// File path or hash
	Needed     int		// Min number of shard needed for reconstruction
	Total      int		// Total number of shards to upload
}

// NewArchonUrl creates an Archon URL from the string representation
func NewArchonUrl(urlString string) (archonUrl *ArchonUrl, err error) {
	if urlString == "" {
		err = errors.New("missing Archon URL")
		return
	}
	if !strings.HasPrefix(urlString, ArcProtocol+"://") {
		err = fmt.Errorf( "should start with %q", ArcProtocol+"://")
		return
	}
	err = errors.New("invalid Archon URL")
	parsedUrl, err2 := url.Parse(urlString)
	if err2 != nil {return}

	if parsedUrl.Path == "" {
		err = errors.Wrap(err,"path is empty")
		return
	}
	archonUrl = new(ArchonUrl)
	archonUrl.Path = parsedUrl.Path
	hostParts := strings.Split(parsedUrl.Host, ".")
	l := len(hostParts)
	var tld string
	switch l {
	case 1:
		// Hash
		tld = hostParts[0]
		if len(tld) > 4 || tld[0] != 'h' {
			err = errors.Wrap(err, "missing h")
			return
		}

	case 3:
		// Named
		archonUrl.Username = hostParts[0]
		if archonUrl.Username == "" {
			err = errors.Wrap(err, "missing user name")
			return
		}
		if err2 := IsLegalUserName(archonUrl.Username); err2 != nil {
			err = errors.Wrap(err, err2.Error())
			return
		}
		perm := UrlPermission(hostParts[1])
		switch perm {
		case Eth, Neo:
			archonUrl.Permission = perm
		default:
			err = errors.Wrap(err, "missing eth or neo")
			return
		}
		tld = hostParts[2]
		if len(tld) > 4 || tld[0] != 'n' {
			err = errors.Wrap(err, "missing .n")
			return
		}

	default: return
	}

	encoding := tld[1:]
	if len(encoding) == 1 {
		// whole file
		if( encoding[0] != '0') {
			err = errors.Wrap(err, "whole file encoding must be 0")
			return
		}
		archonUrl.Needed = 0
		archonUrl.Total = 1
	} else {
		// Sharded
		nums := strings.Split(encoding, ":")
		if len(nums) < 2 {return}
		archonUrl.Needed, _ = strconv.Atoi(nums[0])
		archonUrl.Total, _ = strconv.Atoi(nums[1])
		if archonUrl.Needed <= 0 || archonUrl.Total <= 0 ||archonUrl.Needed > archonUrl.Total {
			err = errors.Wrap(err, "invalid needed or total (0 < needed <= total)")
			return
		}
	}

	if archonUrl.IsHash() {
		// Remove path separator
		archonUrl.Path = strings.ReplaceAll(archonUrl.Path, "/", "")
		_, err = ArchonHashFromString(archonUrl.Path)
	} else {
		err = IsLegalFilePath(archonUrl.Path);
	}
	return
}

func (a *ArchonUrl) Tld() string {
	var s string
	if a.IsHash() {
		s = "h"	// hash
	} else {
		s = "n"	// named
	}
	if a.IsWholeFile() {
		s += "0"
	} else {
		s += fmt.Sprintf( "%d:%d", a.Needed, a.Total)
	}
	return s
}

func (a *ArchonUrl) String() string {
	if a.IsHash() {
		return fmt.Sprintf("%s://%s/%s", ArcProtocol, a.Tld(), a.Path)
	}
	path := a.Path
	if path[0] != '/' {
		path = "/" + path
	}
	return fmt.Sprintf("%s://%s.%s.%s%s", ArcProtocol, a.Username, a.Permission, a.Tld(), path)
}

func (a *ArchonUrl) IsHash() bool {
	return a.Username == ""
}

func (a *ArchonUrl) IsWholeFile() bool {
	return a.Needed == 0
}

func (a *ArchonUrl) DownloadUrl(shardIx int) string {
	url := DownloadEndpoint + "?"
	if shardIx >= 0 {
		url += fmt.Sprintf("shardIdx=%d&", shardIx)
	}
	return url + fmt.Sprintf(`archonUrl=%s`, a)
}

func (a *ArchonUrl) ShardPath(shardIx int) string {
	if a.IsHash() {
		if a.IsWholeFile() {
			return fmt.Sprintf("%s.%s", a.Path, HashFileSuffix)
		}
		return fmt.Sprintf("%s.%d.%s", a.Path, shardIx, HashFileSuffix)
	}
	path := strings.Trim(a.Path, `/\`)
	topFolder := fmt.Sprintf("%s/%s", string(a.Permission), a.Username)
	if a.IsWholeFile() {
		return fmt.Sprintf("%s/%s.%s", topFolder, path, WholeFileSuffix)
	}
	return fmt.Sprintf("%s/%s.%d.%s", topFolder, path, shardIx, ShardFileSuffix)
}

func (a *ArchonUrl) ShardPaths() (paths []string) {
	if a.IsWholeFile() {
		paths = append(paths, a.ShardPath(0))
	} else {
		for i := 0; i < a.Total; i++ {
			paths = append(paths, a.ShardPath(i))
		}
	}
	return
}

/*
func (a *ArchonUrl) PermissionLayerId() permission_layer.PermissionLayerID {
	switch a.Permission {
	case Eth: return permission_layer.EthPermissionId
	case Neo: return permission_layer.NeoPermissionId
	default: return permission_layer.NotPermissionId
	}
}
*/

func IndexFromShardPath(shp string) (index int, err error) {
	parts := strings.Split(shp, ".")
	l := len(parts)
	if l < 3 {
	} else {
		switch parts[l-1] {
		case ShardFileSuffix:
		case HashFileSuffix:
			index, err = strconv.Atoi(parts[l-2])
			if err == nil {
				return
			}
		default:
		}
	}
	err = errors.New("not a shard path")
	return
}
