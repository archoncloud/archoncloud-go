package shards

import (
	"bytes"
	"encoding/binary"
	"fmt"

	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"

	"io"

	permLayer "github.com/archoncloud/archon-dht/permission_layer"
)

const FileContainerCurrentVersion = 0

type FileContainerType uint8

const (
	NoContainer FileContainerType = iota // for non-container upload/download (raw file)
	BrowserOptimized
	ArchiveOptimized // using Infectious RS
)

type EncryptionType uint8

const (
	NoEncryption EncryptionType = iota
)

// in bytes
const CRC32Len = 4
const RedundacyLen = 4

type FileContainer struct {
	Version         uint8 // currently 0
	UploadVersion   permLayer.VersionData
	Type            FileContainerType
	EncryptionType  EncryptionType
	CompressionType uint8 // 0=no compression
	Size            int64 // bytes
	Signature       []byte
	Shards          *ShardsContainer
	Hash            []byte
}

func (fct FileContainerType) HdrLen() int {
	const fileContainerCommonHdrLen = 4
	switch fct {
	case BrowserOptimized:
		return fileContainerCommonHdrLen
	case ArchiveOptimized:
		return fileContainerCommonHdrLen + 1 + CRC32Len // parity+CRC32
	}
	return 0
}

func (fct FileContainerType) TaiLen() int {
	const fileContainerCommonHdrLen = 4
	switch fct {
	case BrowserOptimized:
		return RedundacyLen + ArchonSignatureLen
	case ArchiveOptimized:
		return CRC32Len + ArchonSignatureLen
	}
	return 0
}

func (fct FileContainerType) PayLoadEnd(fcLen int64) int64 {
	tail := fct.TaiLen()
	switch fct {
	case BrowserOptimized:
		return fcLen - int64(tail)
	case ArchiveOptimized:
		payloadAndParity := fcLen - int64(fct.HdrLen()+tail)
		parityLen := DivideRoundUp(uint64(payloadAndParity), 26) // 25-26 parity
		return fcLen - int64(parityLen) - int64(tail)
	}
	return 0
}

// NewFileContainer reads the payload data from r and encodes into s
func NewFileContainer(s *ShardsContainer, versionData *permLayer.VersionData, r io.Reader, account interfaces.IAccount) (*FileContainer, error) {
	fct := BrowserOptimized
	if s.ContainerType == ArchiveOptimizedReedSolomon {
		fct = ArchiveOptimized
	}
	f := FileContainer{
		Version:         FileContainerCurrentVersion,
		UploadVersion:   *versionData,
		Type:            fct,
		EncryptionType:  NoEncryption,
		CompressionType: 0,
		Shards:          s,
	}
	fileContainerBytes, err := f.CreateContainerBytes(r, account)
	if err != nil {
		return nil, err
	}

	s.FileContainerHash = f.Hash
	// This encodes the individual shards
	err = s.Encode(s, fileContainerBytes.Bytes())
	return &f, err
}

// CreateContainerBytes writes the file container data to w. Reads the payload data from r
func (f *FileContainer) CreateContainerBytes(r io.Reader, uploaderAccount interfaces.IAccount) (fileContainerBytes bytes.Buffer, err error) {
	fileContainerBytes = bytes.Buffer{}
	hashWr := NewHashingWriter(&fileContainerBytes)
	hdr := []byte{byte(f.Version), byte(f.Type), byte(f.EncryptionType), byte(f.CompressionType)}
	var numBytes int64 = 0
	n, err := hashWr.Write(hdr)
	if err != nil {
		return
	}
	numBytes += int64(n)

	if f.Type == ArchiveOptimized {
		// Parity for hdr
		p, _ := GenerateRSIParity_4_5(hdr)
		n, _ := hashWr.Write(p)
		numBytes += int64(n)
		n, _ = hashWr.Write(CRC32(fileContainerBytes.Bytes()))
		numBytes += int64(n)
	}
	startOfPayload := numBytes

	// Payload
	nc, err := io.Copy(hashWr, r)
	if err != nil {
		return
	}
	numBytes += nc

	if f.Type == ArchiveOptimized {
		p, _ := GenerateRSIParity_25_26(fileContainerBytes.Bytes()[startOfPayload:])
		_, _ = hashWr.Write(p)
		crc := CRC32(fileContainerBytes.Bytes()[startOfPayload:])
		_, _ = hashWr.Write(crc)
	} else {
		// RedundancyCheck equals the left-most 4 bytes of hash(input), where the input is an 8 byte array consisting of:
		// Version,	Type, EncryptionType, CompressionType and right most 4 bytes of
		// the entire container size (container size % 2^32), big-endian
		Size := numBytes + RedundacyLen + ArchonSignatureLen
		redundancyBytes := [8]byte{byte(f.Version), byte(f.Type), byte(f.EncryptionType), byte(f.CompressionType)}
		binary.BigEndian.PutUint32(redundancyBytes[4:8], uint32(Size&0xFFFF))
		_, err = hashWr.Write(GetArchonHash(redundancyBytes[:])[:RedundacyLen])
		if err != nil {
			return
		}
	}

	sig, err := uploaderAccount.Sign(hashWr.GetHash())
	if err != nil {
		return
	}

	f.Signature = sig

	// Write the signature at the end
	_, err = hashWr.Write(sig)
	f.Hash = hashWr.GetHash()
	return
}

// buf is the file container data. Returns reader of original data
func (s *ShardsContainer) GetOriginalDataFromContainer(container []byte) (io.Reader, error) {
	// Note: container may include trailing zeros that are to be ignored
	hdr, err := s.GetSampleShardHdr()
	if err != nil {
		return nil, err
	}

	containerLen := hdr.GetFileContainerLen()
	if len(container) < int(containerLen) {
		return nil, fmt.Errorf("file container buffer is %d bytes. expected at least %d", len(container), containerLen)
	}
	fct := FileContainerType(container[1])
	if len(container) < fct.HdrLen() {
		return nil, fmt.Errorf("file container buffer is %d bytes. expected at least %d", len(container), containerLen)
	}
	switch fct {
	case BrowserOptimized:
	case ArchiveOptimized:
		// Check header CRC
		crc32 := CRC32(container[:5])
		if bytes.Compare(crc32, container[5:5+CRC32Len]) != 0 {
			return nil, fmt.Errorf("header CRC32 does not match")
		}
	default:
		return nil, fmt.Errorf("unknown file container type: %d", fct)
	}
	originHash := GetArchonHash(container[:containerLen])
	if bytes.Compare(originHash, hdr.GetFileContainerHash()) != 0 {
		return nil, fmt.Errorf("hash does not match original file")
	}

	// TODO: verification - signature, redundancy, CRC32
	r := bytes.NewReader(container[fct.HdrLen():fct.PayLoadEnd(containerLen)])
	return r, nil
}
