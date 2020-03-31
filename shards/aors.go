package shards

import (
	"github.com/vivint/infectious"
	"io"
)

// Archive Optimized Reed-Solomon

func NewAors(total, needed int) *ShardsContainer {
	return &ShardsContainer{
		ArchiveOptimizedReedSolomon,
		RSinfectious,
		total,
		needed,
		nil,
		make([]*Shard, total),
		aorsEncode,
		aorsDecode,
	}
}

// aorsEncode encodes a file container
func aorsEncode(sc *ShardsContainer, fileContainer []byte) (err error) {
	originLen := int64(len(fileContainer))
	f, err := infectious.NewFEC(sc.NumRequired, sc.NumTotal); if err != nil {return}

	// the data to encode must be padded to a multiple of NumRequired
	padding := GetReedSolomonShareSize(originLen, sc.NumRequired)*int64(sc.NumRequired) - originLen
	toEncode := append(fileContainer,make([]byte, padding)...)

	// Prepare to receive the shares of encoded data
	output := func(s infectious.Share) {
		// we need to make a copy of the data. The share data memory gets reused when we return
		h := NewAorsShardHeader(s.Number, sc.NumTotal, sc.NumRequired, originLen, sc.FileContainerHash)
		sc.shards[s.Number] = &Shard{h.ToBytes(), append([]byte(nil), s.Data...)}
	}

	err = f.Encode(toEncode, output)
	return err
}

func aorsDecode(sc *ShardsContainer, writer io.Writer) (err error) {
	var shares []infectious.Share
	for ix, sh := range sc.shards {
		if sh != nil {
			shares = append(shares, infectious.Share{Number: ix, Data: sh.Data})
		}
	}
	f, err := infectious.NewFEC(sc.NumRequired, sc.NumTotal); if err != nil {return}
	fileContainerBytes, err := f.Decode(nil, shares)
	// TODO: Reed Solomon can correct. Perhaps better with more than the required number (?)
	if err != nil {return}
	r, err := sc.GetOriginalDataFromContainer(fileContainerBytes); if err != nil {return}
	_, err = io.Copy(writer, r)
	return
}
