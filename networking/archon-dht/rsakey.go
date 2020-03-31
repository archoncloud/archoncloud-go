package archon_dht

import (
	"crypto/rand"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"sync"

	"github.com/itsmeknt/archoncloud-go/common"
	"github.com/libp2p/go-libp2p-core/crypto"
)

var keyFilePath = struct {
	sync.RWMutex
	s string
}{}

// Generate RSAKey generates a new RSA key based on config.Seed
func GenerateRSAKey(seed int64) (crypto.PrivKey, error) {
	var r io.Reader
	if seed == 1 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(seed))
	}
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	return priv, err
}

// GetRSAKey returns the key from the "rsa_key.priv" file. If needed creates a new key and file
func GetRSAKey(seed int64) (p crypto.PrivKey, e error) {
	keyFilePath.Lock()
	defer keyFilePath.Unlock()
	keyFilePath.s = common.DefaultToExecutable("rsa_key.priv")
	if common.FileExists(keyFilePath.s) {
		data, err := ioutil.ReadFile(keyFilePath.s)
		if err == nil {
			priv, err := crypto.UnmarshalPrivateKey([]byte(data))
			if err == nil {
				return priv, nil
			}
		}
	}
	priv, err := GenerateRSAKey(seed)
	if err == nil {
		data, _ := crypto.MarshalPrivateKey(priv)
		ioutil.WriteFile(keyFilePath.s, data, 0644)
	}
	return priv, err
}
