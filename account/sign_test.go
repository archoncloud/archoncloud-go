package account

import (
	. "github.com/itsmeknt/archoncloud-go/common"
	"gotest.tools/assert"
	"log"
	"testing"
)

func TestSignNeo(t *testing.T) {
	data := make([]byte,1000)
	FillRandom(data)
	hash := GetArchonHash(data)
	accNeo, err := NewNeoAccountFromWif("KydGZaqWyNrT9oN6oBistrmGfX8fzSoGqkAaNZreANeX9uc3PcSA")
	if err != nil {log.Fatalln(err)}
	sig, err := accNeo.Sign(hash)
	if err != nil {log.Fatalln(err)}
	ok := accNeo.Verify(hash,sig, accNeo.EcdsaPublicKeyBytes())
	assert.Equal(t,true, ok)
}
