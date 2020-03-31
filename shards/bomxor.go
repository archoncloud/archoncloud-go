package shards

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"io"
)

// Only supporting mxor_2_6
const BOMxorNumRequired = 2
const BOMxorNumTotal = 6

func NewBOMxor() *ShardsContainer {
	// Currently supporting Mxor_2_6 only
	return &ShardsContainer{
		BrowserOptimizedMxor,
		Mxor_2_6,
		BOMxorNumTotal,
		BOMxorNumRequired,
		nil,
		make([]*Shard, BOMxorNumTotal),
		bomxorEncode,
		bomxorDecode,
	}
}

// Encode encodes a file container
func bomxorEncode(sc *ShardsContainer, fileContainer []byte) error {
	originLen := int64(len(fileContainer))
	if originLen >= 4*humanize.GByte {
		return errors.New("max 4 GByte for mxor")
	}

	// This does the encoding (C code). shardV is a slice of shards (data only)
	shardsV, err := EncodeMxor(fileContainer, sc.erasureCode)
	if err != nil { return err }
	if len(shardsV) != len(sc.shards) {
		return fmt.Errorf("EncodeMxor returned %d shards", len(shardsV))
	}

	for shardIx, shardBuf := range shardsV {
		hdr := NewMxorShardHeader(shardIx, uint32(originLen), sc.FileContainerHash)
		shard := Shard{
			Hdr:	hdr.ToBytes(),
			Data:	shardBuf,
		}
		sc.shards[shardIx] = &shard
	}
	return nil
}

// Decode decodes shards. w writes original data
func bomxorDecode(sc *ShardsContainer, w io.Writer) error {
	// We need 2 (BOMxorNumRequired) shards
	ixs := []int{}
	for i, sh := range sc.shards {
		if sh != nil {
			ixs = append(ixs,i)
			if len(ixs) == 2 {break;}
		}
	}
	if len(ixs) != 2 {
		return fmt.Errorf("two shards are needed")
	}

	fileContainerBytes, err := DecodeMxor(sc.shards[ixs[0]].Data, ixs[0], sc.shards[ixs[1]].Data, ixs[1], sc.erasureCode)
	if err != nil { return err }

	r, err := sc.GetOriginalDataFromContainer(fileContainerBytes)
	if err != nil {return err}

	_, err = io.Copy(w, r)
	return err
}
