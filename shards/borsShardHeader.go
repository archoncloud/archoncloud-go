package shards

import (
	"bytes"
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/pkg/errors"
	"io"
	"strings"
)

// Reed-Solomon encoding. Browser optimized container

const (
	borsCurVersion = 0
)

// This is encoded in the shard header bytes as big-endian
type BorsShardHeader struct {
	Version         byte	// borsCurVersion
	Type            ShardContainerType	// 1=BrowserOptimizedReedSolomon
	ErasureCode 	ErasureCodeType	// RSinfectious
	TotalShardsNum	byte	// Total number of shards
	NeededShardsNum	byte	// Number needed for reconstruction
	ShardIndex      byte
	// The following two refer the FileContainer for the data
	FileContainerDataLen  int64   // whole file container
	FileContainerHash []byte // whole file container
}

func NewBorsShardHeader(index, total, needed uint, fcLen int64, fcHash []byte) *BorsShardHeader {
	h := BorsShardHeader{
		Version:             borsCurVersion,
		Type:              	 BrowserOptimizedReedSolomon,
		ErasureCode:		 RSinfectious,
		TotalShardsNum:      byte(total),
		NeededShardsNum:     byte(needed),
		ShardIndex:          byte(index),
		FileContainerDataLen:fcLen,
		FileContainerHash:   fcHash,
	}
	return &h
}

func NewBorsHeaderFromReader(r io.Reader) (hdr *BorsShardHeader, err error) {
	hdr = new(BorsShardHeader)
	hdr.Version, err = ReadByte(r); if err != nil {return}
	if hdr.Version > borsCurVersion {
		err = fmt.Errorf("V%d or lower supported. Got %d", borsCurVersion, hdr.Version)
		return
	}
	styp, err := ReadByte(r); if err != nil {return}
	if ShardContainerType(styp) != BrowserOptimizedReedSolomon {
		err = errors.New("not bors header")
		return
	}
	hdr.Type = BrowserOptimizedReedSolomon
	b, err := ReadByte(r); if err != nil {return}
	if ErasureCodeType(b) != RSinfectious {
		err = fmt.Errorf("unsupported erasure code: %d", b)
		return
	}
	hdr.ErasureCode = RSinfectious
	hdr.TotalShardsNum, err = ReadByte(r); if err != nil {return}
	hdr.NeededShardsNum, err = ReadByte(r); if err != nil {return}
	hdr.ShardIndex, err = ReadByte(r); if err != nil {return}
	hdr.FileContainerDataLen, err = ReadBigEndianInt64(r); if err != nil {return}
	hdr.FileContainerHash, err = ReadExactly(ArchonHashLen, r); if err != nil {return}
	return
}

func NewBorsShardHeaderFromBytes(data []byte) (*BorsShardHeader, error) {
	return NewBorsHeaderFromReader(bytes.NewReader(data))
}

// ShardHeader interface
func (h *BorsShardHeader) GetContainerType() ShardContainerType {
	return h.Type
}

// ShardHeader interface
func (h *BorsShardHeader) GetErasureCode() ErasureCodeType {
	return h.ErasureCode
}

// ShardHeader interface
func (h *BorsShardHeader) GetNumTotal() int {
	return int(h.TotalShardsNum)
}

// ShardHeader interface
func (h *BorsShardHeader) GetNumRequired() int {
	return int(h.NeededShardsNum)
}

// ShardHeader interface
func (h *BorsShardHeader) GetShardIndex() int {
	return int(h.ShardIndex)
}

// ShardHeader interface
func (h *BorsShardHeader) GetShardPayloadLen() int64 {
	return GetReedSolomonShareSize(h.FileContainerDataLen, int(h.NeededShardsNum))
}

// ShardHeader interface
func (h *BorsShardHeader) GetFileContainerHash() []byte {
	return h.FileContainerHash
}

// ShardHeader interface
func (h *BorsShardHeader) GetFileContainerLen() int64 {
	return h.FileContainerDataLen
}

// ShardHeader interface
func (h *BorsShardHeader) String() string {
	vals := []string{
		fmt.Sprintf("V%d", h.Version),
		fmt.Sprintf("Type=%s", h.Type),
		fmt.Sprintf("Erasure=%s", h.ErasureCode),
		fmt.Sprintf("Encoding=infectious %d/%d", h.NeededShardsNum, h.TotalShardsNum),
		fmt.Sprintf("ShardIndex=%d", h.ShardIndex),
		fmt.Sprintf("FileContainerLen=%d", h.FileContainerDataLen),
	}
	return strings.Join(vals, ", ")
}

// ShardHeader interface
func (h *BorsShardHeader) ToBytes() []byte {
	buf := make([]byte, 0)
	buf = append( buf, byte(h.Version), byte(h.Type), byte(h.ErasureCode),
		byte(h.TotalShardsNum), byte(h.NeededShardsNum), byte(h.ShardIndex))
	buf = append( buf, BigEndianUint64(h.FileContainerDataLen)...)
	buf = append( buf, h.FileContainerHash...)
	return buf
}

// ShardHeader interface
// r provides the payload
func (h *BorsShardHeader) StoreSPShard(w io.Writer, r io.Reader) (hash, uploaderSignature []byte, err error) {
	hw := NewHashingWriter(w)
	hw.Write(h.ToBytes())
	// Copy the payload
	limReader := io.LimitedReader{
		R: r,
		N: GetShardNumBytes(h),
	}
	_, err = io.Copy(hw, &limReader); if err != nil {return}
	hash = hw.GetHash()
	uploaderSignature, err = ReadExactly(ArchonSignatureLen, r)
	return
}
