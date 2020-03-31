package shards

import (
	"bytes"
	"fmt"
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/pkg/errors"
	"io"
	"strings"
)

const MxorCurVersion = 0
const SPHeaderMarker = 0x80

// This is encoded in the shard header bytes as big-endian
type MxorShardHeader struct {
	Version     uint8 // MxorCurVersion
	Type        ShardContainerType // 0=BrowserOptimizedMxor
	Flags       uint8
	ErasureCode ErasureCodeType
	ShardIndex  int
	// The following two refer the FileContainer for the data
	FileContainerDataLen uint32 // Max 4G, including file container
	FileContainerHash    []byte // including file container
}

func NewMxorShardHeader(index int, fcLen uint32, fcHash []byte) *MxorShardHeader {
	h := MxorShardHeader{
		Version: 				MxorCurVersion,
		Type:					BrowserOptimizedMxor,
		Flags:					0,
		ErasureCode: 			Mxor_2_6,
		ShardIndex:           	index,
		FileContainerDataLen:	fcLen,
		FileContainerHash: 		fcHash,
	}
	return &h
}

func NewMxorHeaderFromReader(r io.Reader) (hdr *MxorShardHeader, err error) {
	hdr = new(MxorShardHeader)
	hdr.Version, err = ReadByte(r); if err != nil {return}
	if hdr.Version > MxorCurVersion {
		err = fmt.Errorf("V%d or lower supported. Got %d", MxorCurVersion, hdr.Version)
		return
	}
	styp, err := ReadByte(r); if err != nil {return}
	if ShardContainerType(styp) != BrowserOptimizedMxor {
		err = errors.New("not mxor header")
		return
	}
	hdr.Type = BrowserOptimizedMxor
	hdr.Flags, err = ReadByte(r); if err != nil {return}
	ec, err :=  ReadByte(r); if err != nil {return}
	if ErasureCodeType(ec) != Mxor_2_6 {
		err = fmt.Errorf("unknown erasure code: %d", ec)
		return
	}
	hdr.ErasureCode = Mxor_2_6
	ix, err := ReadByte(r); if err != nil {return}
	hdr.ShardIndex = int(ix)
	hdr.FileContainerDataLen, err = ReadBigEndianUint32(r); if err != nil {return}
	hdr.FileContainerHash, err = ReadExactly(ArchonHashLen,r); if err != nil {return}
	return
}

// ShardHeader interface
func (h *MxorShardHeader) ToBytes() []byte {
	buf := make([]byte, 0)
	buf = append( buf, byte(h.Version), byte(h.Type), byte(h.Flags), byte(h.ErasureCode), byte(h.ShardIndex))
	buf = append( buf, BigEndianUint32(h.FileContainerDataLen)...)
	buf = append( buf, h.FileContainerHash...)
	return buf
}

func NewMxorHeaderFromBytes(data []byte) (*MxorShardHeader, error) {
	return NewMxorHeaderFromReader(bytes.NewReader(data))
}

// ShardHeader interface
func (h *MxorShardHeader) GetContainerType() ShardContainerType {
	return BrowserOptimizedMxor
}

// ShardHeader interface
func (h *MxorShardHeader) GetErasureCode() ErasureCodeType {
	return h.ErasureCode
}

// ShardHeader interface
func (h *MxorShardHeader) GetNumTotal() int {
	return BOMxorNumTotal
}

// ShardHeader interface
func (h *MxorShardHeader) GetNumRequired() int {
	return BOMxorNumRequired
}

// ShardHeader interface
func (h *MxorShardHeader) GetShardIndex() int {
	return h.ShardIndex
}

// ShardHeader interface
func (h *MxorShardHeader) GetShardPayloadLen() int64 {
	return GetShardNumBytes(h)
}

// ShardHeader interface
func (h *MxorShardHeader) GetFileContainerHash() []byte {
	return h.FileContainerHash[:]
}

// ShardHeader interface
func (h *MxorShardHeader) GetFileContainerLen() int64 {
	return int64(h.FileContainerDataLen)
}

// ShardHeader interface
func (h *MxorShardHeader) String() string {
	vals := []string{
		fmt.Sprintf("V%d", h.Version),
		fmt.Sprintf("Type=%s", h.Type),
		fmt.Sprintf("Flags=%d", h.Flags),
		fmt.Sprintf("Erasure=%s", h.ErasureCode),
		fmt.Sprintf("ShardIndex=%d", h.ShardIndex),
		fmt.Sprintf("FileContainerLen=%d", h.FileContainerDataLen),
	}
	return strings.Join(vals, ", ")
}

func MxorShard(r io.Reader) (*Shard, error) {
	// First the header
	hdr, err := NewMxorHeaderFromReader(r)
	if err != nil {return nil, err}

	shard := Shard{
		Hdr: hdr.ToBytes(),
	}

	// Now the data
	dataLen := hdr.GetShardPayloadLen()
	shard.Data = make([]byte, dataLen)
	n, err := r.Read(shard.Data)
	if err != nil {return nil, err}
	if int64(n) != dataLen {
		err = fmt.Errorf("insufficient shard data")
		return nil, err
	}
	return &shard, nil
}

// ShardHeader interface
func (h *MxorShardHeader) StoreSPShard(w io.Writer, r io.Reader) (hash, uploaderSignature []byte, err error) {
	wr := NewHashingWriter(w)
	// Must hash before modifying
	wr.ToHash.Write(h.ToBytes())

	// Now write modified header (just one bit modified)
	h.MarkAsSP()
	_, err = wr.ToData.Write(h.ToBytes()); if err != nil {return}

	// Write (and hash) the shard Data
	limReader := io.LimitedReader{
		R: r,
		N: int64(h.GetShardPayloadLen()),
	}
	_, err = io.Copy(wr,&limReader)
	hash = wr.GetHash()

	uploaderSignature, err = ReadExactly(ArchonSignatureLen, r)
	return
}

func (h *MxorShardHeader) MarkAsSP() {
	h.Flags |= SPHeaderMarker
}
