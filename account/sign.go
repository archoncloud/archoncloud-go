package account

import (
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	ifc "github.com/itsmeknt/archoncloud-go/interfaces"
)

/*
	Private key: 32 bytes
	Public key: 64 bytes
	Address: 20 bytes
*/

const firstPublicKeyByte uint8 = 4 // ecdsa

// Sign and verify is done with ecdsa for both Eth and Neo
func Sign(acc ifc.IAccount, hash []byte) (sig []byte, err error) {
	sig, err = ecrypto.Sign(hash, acc.EcdsaPrivateKey())
	//fmt.Println("----Sign public key: " + account.PublicKeyString())
	//fmt.Println("----Sign hash: " + BytesToString(hash))
	//fmt.Println("----Sign signature: " + BytesToString(sig))
	//fmt.Println("----")
	// For debugging only
	//if !account.Verify(hash, sig) {
	//	panic( "Sign")
	//}
	return
}

func Verify(acc ifc.IAccount, hash, signature, publicKey []byte) bool {
	/*
		fmt.Println("Verify public key: "+BytesToString(pubKey))
		fmt.Println("Verify hash: "+BytesToString(hash))
		fmt.Println("Verify signature: "+signature.String())
		fmt.Println("")
	*/
	if publicKey == nil {
		publicKey = acc.EcdsaPublicKeyBytes()
	}
	signatureNoRecoverID := signature[:len(signature)-1] // remove recovery ID
	// Archon public key stores only last 64 bytes
	ecsdaPubKey := append([]byte{firstPublicKeyByte}, publicKey...)
	return ecrypto.VerifySignature(ecsdaPubKey, hash, signatureNoRecoverID)
}
