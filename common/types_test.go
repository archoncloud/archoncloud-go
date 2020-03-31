package common

import (
	"bytes"
	"encoding/hex"
	"testing"
)

const toHash = "Text to hash. Just an example. Not very long.\nBut still, good enough"

func TestNewArchonHashLength(t *testing.T) {
	hw := NewArchonHash()
	hw.Write([]byte(toHash))
	hash := hw.Sum(nil)
	if len(hash) != ArchonHashLen {
		t.Errorf("Expected hash length of %d, got %d", ArchonHashLen, len(hash))
	}
}

func TestGetArchonHash(t *testing.T) {
	const expectedHashString = "4c2cfd3251f4c0362ac98a4087c4a57303bb74c54dc7698e6ff1b76aaf97db56"
	expectedHash, _ := hex.DecodeString(expectedHashString)
	hash := GetArchonHash([]byte(toHash))
	if !bytes.Equal(hash, expectedHash) {
		t.Error("GetArchonHash returned incorrect hash")
	}
}

func TestArchonHashString(t *testing.T) {
	const expectedString = "68MmsJMtTexAMe9kjx9oQzuAw41cRf7wDFNCEkaDmnxD"
	hash := GetArchonHash([]byte(toHash))
	hashString := ArchonHashString(hash)
	if hashString != expectedString {
		t.Error("ArchonHashString returned incorrect string")
	}
}

func TestArchonHashFromString(t *testing.T) {
	hash := GetArchonHash([]byte(toHash))
	hashString := ArchonHashString(hash)
	hashFromString, err := ArchonHashFromString(hashString)
	if err != nil {t.Error(err)}
	if bytes.Compare(hash, hashFromString) != 0 {
		t.Error("ArchonHashFromString returned a diferent hash")
	}
}
/*
func TestNewContainsEpResponse(t *testing.T) {
	response := `"shards":[0, 1, 10, 11, 2, 3, 4, 5, 6, 7, 8, 9]}`
	r := NewContainsEpResponse([]byte(response))
	if len(r.ShardIdx) != 10 {
		t.Errorf("NewContainsEpResponse wanted %d got %d", 10, len(r.ShardIdx))
		return
	}
}
*/