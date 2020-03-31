package storageProvider

import (
	"bytes"
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"os"
	"strings"
	"time"
)

// pendingUpload records an upload where the file was written, but verification not yet done
type pendingUpload struct {
	Url              *ArchonUrl
	acc              interfaces.IAccount
	TransactionHash  string
	ComputedFileHash []byte
	Signature        []byte
	FilePath         string
	DhtId            string
	Client           string
	Received         time.Time
}

func (pu *pendingUpload) String() string {
	return RelativeToRoot(pu.FilePath)
}

var uploadsPendingChan = make(chan *pendingUpload, 400)

// Pending upload verification results
const (
	verifyUnknown int = iota
	verifyFailed
	verifyOK
)

func registerPendingUpload(pars *uploadPars, client, txHash string, fileHash []byte, filePath string, shardIx int, signature []byte) {
	p := pendingUpload{
		Url: &pars.ArchonUrl,
		acc: pars.acc,
		TransactionHash:  txHash,
		ComputedFileHash: fileHash,
		Signature:        signature,
		FilePath:         filePath,
		DhtId:			  pars.ShardPath(shardIx),
		Client:			  client,
		Received:         time.Now(),
	}
	uploadsPendingChan <- &p
}

// VerifyPendingUploads checks previously uploaded files for validity and removes them if not valid
func VerifyPendingUploads() {
	done := false
	toVerify := make([]*pendingUpload, 0)

	// Runs forever until executable is stopped
	for !done {
		select {
		case p := <-uploadsPendingChan:
			if p == nil {
				// Channel closed. Executable exiting
				done = true
			} else {
				// If we already have the same file (an overwrite not yet verified), remove it
				for i, pe := range toVerify {
					if pe.FilePath == p.FilePath {
						toVerify = append(toVerify[:i], toVerify[i+1:]...)
						break
					}
				}
				toVerify = append(toVerify, p)
			}

		case <-time.After(7*time.Second):
			if len(toVerify) == 0 {break}

			remaining := make([]*pendingUpload,0)
			for _, pu := range toVerify {
				diff := time.Since(pu.Received)
				switch status, msg := pu.Verify(); status {
				case verifyOK:
					parts := strings.Split(pu.DhtId,"/")
					err := AnnounceToDht(pu.DhtId, parts[0])
					if err != nil {
						LogError.Printf("%s could not be registered with DHT (%s)\n", pu, err.Error())
					} else {
						LogInfo.Printf("Upload of %s verified after %s\n", pu, diff)
					}
				case verifyUnknown:
					if diff > 10 * time.Minute {
						LogError.Printf("%q could not be validated after %s. Removed\n",
							pu, diff)
						_ = os.Remove(pu.FilePath)
					} else {
						// Need to try again later
						remaining = append(remaining, pu)
						LogDebug.Printf("Waiting for verification of %s\n", pu)
					}
				case verifyFailed:
					LogError.Printf("%q is not a valid upload\n%s\nRemoved\n", pu, msg)
					_ = os.Remove(pu.FilePath)
				}
			}
			LogDebug.Printf("Remaining to verify: %d\n", len(remaining))
			toVerify = remaining
		}
	}
}

func (p *pendingUpload) Verify() (int, string) {
	if !p.acc.IsTxAccepted(p.TransactionHash) {
		// Not yet processed. Do later
		return verifyUnknown, ""
	}
	upTx, err := p.acc.GetUploadTxInfo(p.TransactionHash)
	if err != nil {
		return verifyFailed, err.Error()
	}
	valid := false
	thisSP := p.acc.AddressBytes()
	for _, validUploader := range upTx.SPs {
		if bytes.Equal(thisSP, validUploader) {
			valid = true
			break
		}
	}
	if !valid {
		return verifyFailed, "this SP is not in the upload list"
	}
	hash := p.ComputedFileHash
	signature := p.Signature
	if !p.acc.Verify(hash, signature, upTx.PublicKey) {
		return verifyFailed, fmt.Sprintf("signature does not match for\n   key: %s\n   hash: %s\n   signature: %s",
			BytesToString(upTx.PublicKey), BytesToString(hash), BytesToString(signature))
	}
	a := ActivityInfo{
		Upload:   true,
		Url:      p.Url,
		Client:   p.Client,
		NumBytes: FileSize(p.FilePath),
	}
	RecordActivity(&a)
	return verifyOK, ""
}
