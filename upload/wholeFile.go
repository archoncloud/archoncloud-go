package upload

import (
	"fmt"
	"github.com/archoncloud/archoncloud-go/account"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/shards"
	"io"
	"mime/multipart"
	"os"
)

// wholeFileUpload uploads the whole file as is, no container, no shards
// If more than one SP is provided, it will upload to all
func (u *Request) wholeFileUpload(a *ArchonUrl, sps StorageProviders) (price int64, err error) {
	file, err := os.Open(u.FilePath)
	if err != nil {return}
	fmt.Printf("Computing signature\n")
	hashString, uploaderSignature, size, err := account.ArchonSignatureFor(file, u.UploaderAccount)
	file.Close()
	if err != nil {return}

	if a.IsHash() {
		a.Path = ArchonHashString(StringToBytes(hashString))
	}

	// There will be no container generated, but we need the data from it
	fileContainer := shards.FileContainer{
		Version:         shards.FileContainerCurrentVersion,
		Type:            shards.NoContainer,
		EncryptionType:  shards.NoEncryption,
		CompressionType: 0,
		Size:            size,
		Signature:       uploaderSignature,
	}
	ix := RandomInt(0, len(sps))
	sp := sps[ix]
	// New proposed upload transaction
	txid, price, err := account.ProposeUpload(u.UploaderAccount, &fileContainer, nil, a, []SpProfile{sp}, u.MaxPayment)
	if err != nil {return}
	fmt.Println("Upload transaction ID:", txid)

	url := sp.Url(u.PreferHttp)
	fmt.Printf("Uploading to: %s\n", url)
	bp := NewByteProgress("Upload", uint64(fileContainer.Size))
	err = u.postWholeFile(url, txid, a, bp)
	bp.End()
	return
}

// postWholeFile posts a whole file to an SP. This can run in parallel
func (u *Request) postWholeFile(spUrl, txid string, aurl *ArchonUrl, bp *ByteProgress) error {
	// Use a pipe so we don't need the whole file in memory
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	var writeErr error
	go func() {
		defer w.Close()
		defer m.Close()
		// Important: CreateFormFile must be in go function
		part, writeErr := m.CreateFormFile(UploadFileKey, u.FilePath)
		if writeErr != nil {return}

		file, writeErr := os.OpenFile(u.FilePath, os.O_RDONLY, 0)
		if writeErr != nil {return}

		defer file.Close()
		_, writeErr = io.Copy(part, file)
	}()

	targetUrl := u.TargetUrl(spUrl, txid)
	err := PostFromReaderWithProgress(targetUrl, r, m.FormDataContentType(), bp)
	if err == nil {err = writeErr}
	return err
}
