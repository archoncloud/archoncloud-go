package storageProvider

import (
	"fmt"
	"github.com/dustin/go-humanize"
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/itsmeknt/archoncloud-go/interfaces"
	"github.com/itsmeknt/archoncloud-go/shards"
	"github.com/pkg/errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	fp "path/filepath"
	"strings"
)

type uploadPars struct {
	ArchonUrl
	acc       interfaces.IAccount
	overwrite bool
}

// uploadHandler responds to upload requests (/upload endpoint)
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Note: the upload transaction at this stage may be pending
	// We will save the files regardless and verify later
	pars := uploadPars{}
	transactionHash := r.URL.Query().Get(TransactionHashQuery)
	if transactionHash == "" {
		httpBadRequest(w, r, errors.New("missing transaction hash in URL"))
		return
	}
	blockCh := strings.ToLower(r.URL.Query().Get(ChainQuery))
	switch blockCh {
	case "eth":
		pars.acc = GetAccount(interfaces.EthAccountType)
	case "neo":
		pars.acc = GetAccount(interfaces.NeoAccountType)
	default:
		httpBadRequest(w, r, fmt.Errorf("missing or unknown " + ChainQuery))
		return
	}
	if pars.acc == nil {
		httpBadRequest(w, r, fmt.Errorf("can't handle uploads for chain " + ChainQuery))
		return
	}
	upTx, err := pars.acc.GetUploadTxInfo(transactionHash)
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}

	if err = r.ParseMultipartForm(64 * humanize.MByte); err != nil {
		httpBadRequest(w, r, err)
		return
	}

	// FormFile returns the first file for the given key
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	reader, handler, err := r.FormFile(UploadFileKey)
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}
	defer reader.Close()

	cloudDir := r.URL.Query().Get(CloudDir)
	if cloudDir != "" {
		cloudDir, _ = url.QueryUnescape(cloudDir)
	}
	fileName, err := getFileName(handler,cloudDir)
	if err != nil {
		httpBadRequest(w, r, err)
		return
	}

	hashUrl := BoolFromQuery(HashUrlQuery, r)
	if !hashUrl {
		if upTx.UserName == "" {
			httpBadRequest(w, r, errors.New("user name is missing in transaction"))
			return
		}
	}

	pars.ArchonUrl = ArchonUrl{
		upTx.UserName,
		pars.acc.Permission(),
		fileName,
		0,
		0,
	}
	pars.overwrite = BoolFromQuery(OverwriteQuery, r)
	LogTrace.Printf( "%s: Starting upload of %s", requestInfo(r), fileName)
	if upTx.FileContainerType == uint8(shards.NoContainer) {
		err = uploadWholeFile(w, r, reader, &pars, upTx)
	} else {
		err = uploadShard(w, r, reader, &pars, upTx)
	}
	if err != nil {
		httpErr(w, r, err, http.StatusForbidden)
	}
}

func uploadWholeFile(w http.ResponseWriter, r *http.Request, reader multipart.File, pars *uploadPars, upTx *interfaces.UploadTxInfo) error {
	tempFile, err := GetTempFile()
	if err != nil {return err}

	hashWriter := NewHashingWriter(tempFile)
	_, err = io.Copy(hashWriter, reader)
	tempFile.Close()
	defer func() {
		os.Remove(tempFile.Name())
	}();

	if err != nil {return err}
	hash := hashWriter.GetHash()
	var shardPath string
	if pars.IsHash() {
		pars.Path = ArchonHashString(hash)
	}
	shardPath = StoredFilePath(&pars.ArchonUrl, 0)
	if !pars.overwrite && FileExists(shardPath) {
		return errors.New("shard already exists")
	}
	err = os.MkdirAll(fp.Dir(shardPath), os.ModeDir|os.ModePerm)
	if err != nil {return err}

	err = os.Rename(tempFile.Name(), shardPath)
	if err != nil {return err}

	registerPendingUpload(pars, r.RemoteAddr, upTx.TxId, hash, shardPath, 0, upTx.Signature)
	httpInfo(w, r, fmt.Sprintf("Uploaded to %s (%d). Pending verification", fp.Base(shardPath), FileSize(shardPath)))
	return nil
}

func uploadShard(w http.ResponseWriter, r *http.Request, reader multipart.File, pars *uploadPars, upTx *interfaces.UploadTxInfo) error {
	// Process the shard header first
	hdr, err := shards.NewShardHeader(reader); if err != nil {return err}
	LogTrace.Printf("Header=%s\n", hdr.String())

	pars.Needed = hdr.GetNumRequired()
	pars.Total = hdr.GetNumTotal()
	shardFile, err := getShardFile(pars, hdr)
	if err != nil {return err}
	uploadedPath := shardFile.Name()
	defer func() {
		if shardFile != nil {
			// Clean up on error
			shardFile.Close()
			os.Remove(shardFile.Name())
			shardFile = nil
		}
	}()

	hash, uploaderSignature, err := hdr.StoreSPShard(shardFile,reader)
	if err != nil {return err}

	// Uploader signature will be verified later
	_, _ = shardFile.Write(uploaderSignature)

	// Add the SP signature
	spSignature, err := pars.acc.Sign(GetArchonHash(uploaderSignature))
	if err != nil {return err}

	_, _ = shardFile.Write(spSignature)
	err = shardFile.Close(); if err != nil {return err}
	shardFile = nil // To prevent the file being removed by "defer"

	registerPendingUpload(pars, r.RemoteAddr, upTx.TxId, hash, uploadedPath, hdr.GetShardIndex(), uploaderSignature)
	httpInfo(w, r, fmt.Sprintf("Uploaded to %s (%d). Pending verification", fp.Base(uploadedPath), FileSize(uploadedPath)))
	return nil
}

func getShardFile(pars *uploadPars, hdr shards.ShardHeader) (shardFile *os.File, err error) {
	if pars.IsHash() {
		pars.Path = ArchonHashString(hdr.GetFileContainerHash())
	}
	shardPath := StoredFilePath(&pars.ArchonUrl, hdr.GetShardIndex())
	if !pars.overwrite && FileExists(shardPath) {
		err = errors.New("shard already exists")
	}
	err = os.MkdirAll(fp.Dir(shardPath), os.ModeDir|os.ModePerm)
	if err == nil {
		shardFile, err = os.Create(shardPath)
	}
	return
}

func getFileName(handler *multipart.FileHeader, cloudDir string) (fileName string, err error) {
	fileName = strings.TrimSpace(handler.Filename)
	if fileName == "" {
		err = errors.New("file path is empty")
		return
	}
	// The below is needed if we are on Linux and get the upload request from Windows
	if len(fileName) >= 2 && fileName[1] == ':' {
		// Windows drive letter
		fileName = fileName[2:]
	}
	// Adapt to running OS
	switch os.PathSeparator {
	case '/': fileName = strings.ReplaceAll(fileName, `\`, `/`)
	case '\\': fileName = strings.ReplaceAll(fileName, `/`, `\`)
	}
	fileName = fp.Base(fileName)
	c := strings.TrimSpace(cloudDir)
	if c != "" {
		fileName = fp.Join(c, fileName)
	}
	err = IsLegalFilePath(fileName)
	return
}
