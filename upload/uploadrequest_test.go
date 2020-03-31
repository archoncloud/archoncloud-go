package upload

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/archoncloud/archoncloud-go/account"
	"github.com/archoncloud/archoncloud-go/blockchainAPI/ethereum/client_utils"
	. "github.com/archoncloud/archoncloud-go/common"
	"testing"
)

func TestUploadSignature(t *testing.T) {
	account, err := account.GetTestAccount()
	if err != nil {
		t.Error(err)
		return
	}
	uplPar := client_utils.UploadParams{
		SigningKey:         account.PrivateKeyBytes,
		Address:            account.Address,
		ServiceDuration:    0,
		MinSLARequirements: 0,
		MaxBidPrice:        0,
		ArchonFilepath:     "tmp/file3.txt",
		Filesize:           2100,
		FileContainerType:  0,
		EncryptionType:     0,
		CompressionType:    0,
		ShardContainerType: 0,
		ErasureCodeType:    0,
		CustomField:        0,
		ContainerSignature: account.ArchonSignature{},
		SPsToUploadTo:      nil,
	}
	txId, err := client_utils.ProposeUpload(&uplPar)
	if err != nil {
		t.Error(err)
		return
	}

	txHash, _ := hexutil.Decode(txId)
	var txHashA [32]byte
	copy(txHashA[:], txHash)
	upTx, err := uploadtx.GetUploadTx(txHashA)
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal( upTx.PublicKey[:], account.PublicKeyBytes[:]) {
		up := BytesToString(upTx.PublicKey[:])
		ac := account.PublicKeyString()
		t.Errorf("GetUploadTx returned a different public key.\n   got:  %s\n   want: %s", up, ac)
	}
}
