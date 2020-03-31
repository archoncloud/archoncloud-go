package shards

import (
	"bytes"
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/pkg/errors"
	"hash/crc32"
	"io"
)

type ErasureCodeType int8

const (
	ErasureCodeNone ErasureCodeType = -1
	Mxor_2_3        ErasureCodeType = 0
	Mxor_2_4        ErasureCodeType = 1
	Mxor_2_6        ErasureCodeType = 2	// Only this mxor is currently implemented (no plan for the others)
	Mxor_2_7        ErasureCodeType = 3
	RSinfectious    ErasureCodeType = 4 // Reed-Solomon using infectious package
)

func (e ErasureCodeType) String() string {
	switch e {
	case Mxor_2_3: return "mxor_2_3"
	case Mxor_2_4: return "mxor_2_4"
	case Mxor_2_6: return "mxor_2_6"
	case Mxor_2_7: return "mxor_2_7"
	case RSinfectious: return "RSinfectious"
	default: return "?"
	}
}

type ShardContainerType int8

const (
	ShardContainerNone			ShardContainerType = -1
	BrowserOptimizedMxor		ShardContainerType = 0
	BrowserOptimizedReedSolomon ShardContainerType = 1
	ArchiveOptimizedReedSolomon ShardContainerType = 2
)

func (s ShardContainerType) String() string {
	switch s {
	case BrowserOptimizedMxor: return "bo_mxor"
	case BrowserOptimizedReedSolomon: return "bo_RS"
	case ArchiveOptimizedReedSolomon: return "ao_RS"
	default: return "?"
	}
}

type ShardHeader interface {
	GetContainerType() ShardContainerType
	GetErasureCode() ErasureCodeType
	GetNumTotal() int
	GetNumRequired() int
	GetShardIndex() int
	GetShardPayloadLen() int64   // just shard payload, no header, no parity, no CRC

	// file container, same for all shards
	GetFileContainerLen() int64	// the whole file container
	GetFileContainerHash() []byte

	String() string
	ToBytes() []byte
	// r is the payload (shard data). To be used on the SP
	StoreSPShard(w io.Writer, r io.Reader) (hash, uploaderSignature []byte, err error)
}

func GetShardDataLen(fileLen int64, shardContainerType ShardContainerType, numRequired int) (ln int64, err error) {
	// payload plus parity or CRC when needed (up to signature), but does not include header
	switch shardContainerType {
	case BrowserOptimizedMxor:
		// numRequired is fixed
		numTotal := BOMxorNumTotal
		numRequired = BOMxorNumRequired
		rounded := RoundUp(uint64(fileLen), uint64(numTotal))
		// Note: This only works when numNeeded is a divisor or numTotal, which is the case now
		ln = int64(rounded) / int64(numRequired);

	case BrowserOptimizedReedSolomon:
		ln = GetReedSolomonShareSize(fileLen, numRequired)

	case ArchiveOptimizedReedSolomon:
		// payload is followed by parity and CRC
		l := GetReedSolomonShareSize(fileLen, numRequired)
		ln =  l + GetReedSolomonShareSize(l, aorsCorrData) + 4

	default:
		err = errors.New( "unknown erasure code")
	}
	return
}

func GetShardTotalLen(fileLen int64, shardContainerType ShardContainerType, numRequired int) (ln int64, err error) {
	// Includes everything
	ln, err = GetShardDataLen(fileLen, shardContainerType, numRequired)
	if err != nil {return}
	// Add header length
	switch shardContainerType {
	case BrowserOptimizedMxor:
		ln += 41
	case BrowserOptimizedReedSolomon:
		ln += 46
	case ArchiveOptimizedReedSolomon:
		ln += 55
	}
	// Add uploader signature length
	ln += int64(ArchonSignatureLen)
	return
}

func GetShardNumBytes(h ShardHeader) int64 {
	// payload plus parity or CRC when needed (up to signature), but not header
	l, _ := GetShardDataLen(h.GetFileContainerLen(), h.GetContainerType(), h.GetNumRequired())
	return l
}

type Shard struct {
	Hdr		[]byte	// shard header
	Data	[]byte	// shard data
}

func (shard *Shard) Len() int {
	return len(shard.Hdr)+len(shard.Data)
}

func (shard *Shard) GetContainerType() ShardContainerType {
	if len(shard.Hdr) > 2 {
		switch ShardContainerType(shard.Hdr[1]) {
		case BrowserOptimizedMxor, BrowserOptimizedReedSolomon, ArchiveOptimizedReedSolomon:
			return ShardContainerType(shard.Hdr[1])
		}
	}
	return ShardContainerNone
}

// WriteShardContainer writes the whole container to w
func (shard *Shard) WriteShardContainer(w io.Writer, acc interfaces.IAccount) (err error) {
	// This builds the hash and writes the Data
	hw := NewHashingWriter(w)
	_, err = hw.Write(shard.Hdr); if err != nil {return}
	_, err = io.Copy(hw, bytes.NewBuffer(shard.Data)); if err != nil {return}
	if shard.GetContainerType() == ArchiveOptimizedReedSolomon {
		// Copy the parity portion
		parity, err := GenerateRSIParity(shard.Data, aorsCorrData, aorsCorrParity); if err != nil {return err}
		_, err = io.Copy(hw, bytes.NewReader(parity[0]) ); if err != nil {return err}
		expandedData := append(shard.Data, parity[0]...)
		computedCrc := crc32.ChecksumIEEE(expandedData)
		hw.Write(BigEndianUint32(computedCrc))
	}
	hash := hw.GetHash()
	uploaderSignature, err := acc.Sign(hash)
	if err == nil {
		_, err = w.Write(uploaderSignature)
	}
	return
}

// sanity check
func (shard *Shard) VerifyShardDataSize() error {
	hdr, err := shard.GetShardHeader()
	if err != nil {return err}

	expectedDataSize := hdr.GetShardPayloadLen()
	dataSize := len(shard.Data)
	if expectedDataSize != int64(dataSize) {
		return fmt.Errorf("shard size is %d. Expected %d", dataSize, expectedDataSize )
	}
	return nil
}

func NewShardHeader(r io.ReadSeeker) (shdr ShardHeader, err error) {
	_, err = ReadByte(r); if err != nil {return}
	containerType, err := ReadByte(r); if err != nil {return}
	Rewind(r)
	switch ShardContainerType(containerType) {
	case BrowserOptimizedMxor:
		shdr, err = NewMxorHeaderFromReader(r)
	case BrowserOptimizedReedSolomon:
		shdr, err = NewBorsHeaderFromReader(r)
	case ArchiveOptimizedReedSolomon:
		shdr, err = NewAorHeaderFromReader(r)
	default:
		err = fmt.Errorf("unknown shard container type %d", containerType)
	}
	return
}

func (shard *Shard) GetShardHeader() (hdr ShardHeader, err error) {
	return NewShardHeader(bytes.NewReader(shard.Hdr))
}

func NewShardFromSP(shardBytes []byte) (sh *Shard, err error) {
	hdr, err := NewShardHeader(bytes.NewReader(shardBytes)); if err != nil {return}
	sh = new(Shard)
	sh.Hdr = hdr.ToBytes()
	l := len(sh.Hdr)
	sh.Data = shardBytes[l:l+int(hdr.GetShardPayloadLen())]
	return
}

// ShardsContainer contains all the shards
type ShardsContainer struct {
	// Not thread safe. Should not be shared between go routines
	ContainerType     ShardContainerType
	erasureCode       ErasureCodeType
	NumTotal          int    // Total number of shards
	NumRequired       int    // Min number required for reconstruction
	FileContainerHash []byte // including file container
	shards            []*Shard

	// Encode encodes a file container (to shards)
	Encode			func (sc *ShardsContainer, fileContainer []byte) error
	// Decode decodes from shards to original data, discarding file container
	Decode			func (sc *ShardsContainer, writer io.Writer) error
}

func NewShardsContainer(shardBytes []byte) (s *ShardsContainer, err error) {
	r := bytes.NewReader(shardBytes)
	hdr, err := NewShardHeader(r)
	if err != nil {return}

	switch hdr.GetContainerType() {
	case BrowserOptimizedMxor:
		s = NewBOMxor()
	case BrowserOptimizedReedSolomon:
		s = NewBors(hdr.GetNumTotal(), hdr.GetNumRequired())
	case ArchiveOptimizedReedSolomon:
		s = NewAors(hdr.GetNumTotal(), hdr.GetNumRequired())
	default:
		err = fmt.Errorf("unknown container type: %d", hdr.GetContainerType())
	}
	return
}

func (sc *ShardsContainer) GetContainerType() ShardContainerType {
	return sc.ContainerType
}

func (sc *ShardsContainer) GetErasureCode() ErasureCodeType {
	return sc.erasureCode
}

func (sc *ShardsContainer) GetShard(index int) *Shard {
	if index >= len(sc.shards) {
		return nil
	}
	return sc.shards[index]
}

func (sc *ShardsContainer) SetShard(index int, shard *Shard) error {
	if index >= len(sc.shards) {
		return errors.New("shard index too large")
	}
	sc.shards[index] = shard
	return nil
}

// GetFirstShard returns first existing shards, nil if there are none
func (sc *ShardsContainer) GetFirstShard() *Shard {
	for _, sh := range sc.shards {
		if sh != nil {
			return sh
		}
	}
	return nil
}

// GetShardPayloadLen returns shard data size (without header). All shards have same size
func (sc *ShardsContainer) GetShardDataLen() int64 {
	sh := sc.GetFirstShard()
	if sh == nil {
		return 0
	}
	hdr, _ := NewShardHeader(bytes.NewReader(sh.Hdr))
	return hdr.GetShardPayloadLen()
}

// GetShardNumBytes returns the numbers of bytes in the shard
func (sc *ShardsContainer) GetShardNumBytes() int64 {
	sh := sc.GetFirstShard()
	if sh == nil {
		return 0
	}
	hdr, err := NewShardHeader(bytes.NewReader(sh.Hdr))
	if err != nil {return 0}
	return GetShardNumBytes(hdr)
}

func (sc *ShardsContainer) GetOriginDataHashString() string {
	return ArchonHashString(sc.FileContainerHash)
}

func (sc *ShardsContainer) GetOriginDataHash() *ArchonHashBytes {
	var hb ArchonHashBytes
	copy(hb[:], sc.FileContainerHash)
	return &hb
}

func (sc *ShardsContainer) GetNumShards() int {
	return len(sc.shards)
}

func (sc *ShardsContainer) HaveAllNeededShards() bool {
	numNeeded := sc.NumRequired
	for _, sh := range sc.shards {
		if sh != nil {
			numNeeded--
			if numNeeded <= 0 {return true}
		}
	}
	return numNeeded <= 0
}

// GetSampleShardHdr returns the header of the first shard. Useful when needing info
// that is common to all shards
func (sc *ShardsContainer) GetSampleShardHdr() (hdr ShardHeader, err error) {
	shard := sc.GetFirstShard()
	if shard == nil {
		err = fmt.Errorf("no shards")
	} else {
		hdr, err = NewShardHeader(bytes.NewReader(shard.Hdr))
	}
	return
}
