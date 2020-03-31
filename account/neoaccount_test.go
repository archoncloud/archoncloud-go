package account

import "testing"

func TestNewNeoAccount(t *testing.T) {
	a, err := NewNeoAccount(`C:\Archon\archoncloud-contracts\neo\Wallets\sp1.json`, "sp1")
	if err != nil {t.Fatal(err)}

	if a.PrivateKeyString() == "" {t.Fatal("no private key")}
}
