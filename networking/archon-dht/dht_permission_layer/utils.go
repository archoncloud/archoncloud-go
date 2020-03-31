package dht_permission_layer

import (
	"sync"
)

type SafeFilename struct {
	sync.RWMutex
}

var SpFilenames = struct {
	sync.RWMutex
	M map[string]*SafeFilename
}{M: make(map[string]*SafeFilename)}
