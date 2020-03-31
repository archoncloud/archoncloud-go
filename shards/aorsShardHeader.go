package shards

import (
	"bytes"
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/pkg/errors"
	"hash/crc32"
	"io"
	"strings"
)

// Reed-Solomon encoding. Archive optimized container

const (
	aorsCurVersion = 0
	aorsHdrParityLen = 3
	aorsCorrData = 25		// 26. 26 for generating one parity shard
	aorsCorrParity = 1
)

// This is encoded in the shard header bytes as big-endian
type AorsShardHeader struct {
	Version         byte	// aorsCurVersion
	Type            ShardContainerType	// 2=ArchiveOptimizedReedSolomon
	ErasureCode 	ErasureCodeType	// RSinfectious
	TotalShardsNum	int	// Total number of shards
	NeededShardsNum	int	// Number needed for reconstruction
	ShardIndex      int
	// The following two refer the FileContainer for the data
	FileContainerLen  int64  // whole file container
	FileContainerHash []byte // whole file container
}

func NewAorsShardHeader(index, total, needed int, fcLen int64, fcHash []byte) *AorsShardHeader {
	h := AorsShardHeader{
		Version:           aorsCurVersion,
		Type:              ArchiveOptimizedReedSolomon,
		ErasureCode:       RSinfectious,
		TotalShardsNum:    total,
		NeededShardsNum:   needed,
		ShardIndex:        index,
		FileContainerLen:  fcLen,
		FileContainerHash: fcHash,
	}
	return &h
}

// ShardHeader interface
func (h *AorsShardHeader) GetContainerType() ShardContainerType {
	return ArchiveOptimizedReedSolomon
}

// ShardHeader interface
func (h *AorsShardHeader) GetErasureCode() ErasureCodeType {
	return h.ErasureCode
}

// ShardHeader interface
func (h *AorsShardHeader) GetNumTotal() int {
	return h.TotalShardsNum
}

// ShardHeader interface
func (h *AorsShardHeader) GetNumRequired() int {
	return h.NeededShardsNum
}

// ShardHeader interface
func (h *AorsShardHeader) GetShardIndex() int {
	return h.ShardIndex
}

// ShardHeader interface
func (h *AorsShardHeader) GetShardPayloadLen() int64 {
	return GetReedSolomonShareSize(h.FileContainerLen, h.NeededShardsNum)
}

// ShardHeader interface
func (h *AorsShardHeader) GetFileContainerHash() []byte {
	return h.FileContainerHash
}

// ShardHeader interface
func (h *AorsShardHeader) GetFileContainerLen() int64 {
	return h.FileContainerLen
}

// ShardHeader interface
func (h *AorsShardHeader) String() string {
	vals := []string{
		fmt.Sprintf("V%d", h.Version),
		fmt.Sprintf("Type=%s", h.Type),
		fmt.Sprintf("Erasure=%s", h.ErasureCode),
		fmt.Sprintf("Encoding=infectious %d/%d", h.NeededShardsNum, h.TotalShardsNum),
		fmt.Sprintf("ShardIndex=%d", h.ShardIndex),
		fmt.Sprintf("FileContainerLen=%d", h.FileContainerLen),
	}
	return strings.Join(vals, ", ")
}

// ShardHeader interface. Generates the byte needed for staorage
func (h *AorsShardHeader) ToBytes() []byte {
	buf := make([]byte, 0)
	buf = append( buf, byte(h.Version), byte(h.Type), byte(h.ErasureCode))
	buf = append( buf, BigEndianUint32(uint32(h.TotalShardsNum))...)
	buf = append( buf, BigEndianUint32(uint32(h.NeededShardsNum))...)
	buf = append( buf, BigEndianUint32(uint32(h.ShardIndex))...)
	buf = append( buf, BigEndianUint64(h.FileContainerLen)...)
	buf = append( buf, h.FileContainerHash...)
	p, err := GenerateRSIParity(buf,aorsCorrData,aorsCorrParity)
	if err != nil || len(p) != 1 || len(p[0]) != aorsHdrParityLen {
		panic("AorsShardHeader ToBytes")
	}
	buf = append(buf, p[0]...)
	buf = append(buf, CRC32(buf)...)
	return buf
}

// ShardHeader interface
// r provides the shard data
func (h *AorsShardHeader) StoreSPShard(w io.Writer, r io.Reader) (hash, uploaderSignature []byte, err error) {
	hw := NewHashingWriter(w)
	hw.Write(h.ToBytes())
	// Copy the payload
	// Write (and hash) the shard Data
	limReader := io.LimitedReader{
		R: r,
		N: GetShardNumBytes(h),
	}
	_, err = io.Copy(hw,&limReader); if err != nil {return}
	hash = hw.GetHash()
	uploaderSignature, err = ReadExactly(ArchonSignatureLen, r)
	return
}

func aorsSHFromReader(dataR io.Reader) (hdr *AorsShardHeader, hdrBytes []byte, err error) {
	var b bytes.Buffer
	r := io.TeeReader(dataR,&b)
	hdr = new(AorsShardHeader)
	hdr.Version, err = ReadByte(r); if err != nil {return}
	if hdr.Version > aorsCurVersion {
		err = fmt.Errorf("V%d or lower supported. Got %d", aorsCurVersion, hdr.Version)
		return
	}
	styp, err := ReadByte(r); if err != nil {return}
	if ShardContainerType(styp) != ArchiveOptimizedReedSolomon {
		err = errors.New("not aors header")
		return
	}
	hdr.Type = ArchiveOptimizedReedSolomon
	ec, err :=  ReadByte(r); if err != nil {return}
	if ErasureCodeType(ec) != RSinfectious {
		err = errors.New("unknown erasure code")
		return
	}
	hdr.ErasureCode = RSinfectious
	tot, err := ReadBigEndianInt32(r); if err != nil {return}
	hdr.TotalShardsNum = int(tot)
	needed, err := ReadBigEndianInt32(r); if err != nil {return}
	hdr.NeededShardsNum = int(needed)
	index, err := ReadBigEndianInt32(r); if err != nil {return}
	hdr.ShardIndex = int(index)
	hdr.FileContainerLen, err = ReadBigEndianInt64(r); if err != nil {return}
	hdr.FileContainerHash, err = ReadExactly(ArchonHashLen, r); if err != nil {return}
	_, err = ReadExactly(aorsHdrParityLen, r); if err != nil {return}
	hdrBytes = b.Bytes()
	return
}

func NewAorHeaderFromReader(dataR io.Reader) (h *AorsShardHeader, err error) {
	h, hdrB, err := aorsSHFromReader(dataR)
	if err != nil {return}

	computedCrc := crc32.ChecksumIEEE(hdrB)
	crc, err := ReadBigEndianUint32(dataR); if err != nil {return}
	if crc == computedCrc {return}

	// Try to correct using parity
	sep := len(hdrB) - aorsHdrParityLen
	hdrData := hdrB[:sep]
	hdrParity := hdrB[sep:]
	correctedHdr, err := CorrectRSI(hdrData, [][]byte{hdrParity},aorsCorrData,aorsCorrParity)
	if err != nil {
		err = errors.New("could not correct aors header")
		return
	}
	h, _, err = aorsSHFromReader(bytes.NewReader(correctedHdr))
	return
}

