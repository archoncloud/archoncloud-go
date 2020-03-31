package shards

import (
	"github.com/vivint/infectious"
	"io"
)

// Browser Optimized Reed-Solomon

func NewBors(total, needed int) *ShardsContainer {
	return &ShardsContainer{
		BrowserOptimizedReedSolomon,
		RSinfectious,
		total,
		needed,
		nil,
		make([]*Shard, total),
		borsEncode,
		borsDecode,
	}
}

// borsEncode encodes a file container
func borsEncode(sc *ShardsContainer, fileContainer []byte) error {
	originLen := int64(len(fileContainer))
	f, err := infectious.NewFEC(sc.NumRequired, sc.NumTotal)
	if err != nil {return err}

	// the data to encode must be padded to a multiple of NumRequired
	padding := GetReedSolomonShareSize(originLen, sc.NumRequired)*int64(sc.NumRequired) - originLen
	toEncode := append(fileContainer,make([]byte, padding)...)

	// Prepare to receive the shares of encoded data
	output := func(s infectious.Share) {
		// we need to make a copy of the data. The share data memory gets reused when we return
		h := NewBorsShardHeader(uint(s.Number), uint(sc.NumTotal), uint(sc.NumRequired), originLen, sc.FileContainerHash)
		sc.shards[s.Number] = &Shard{h.ToBytes(), append([]byte(nil), s.Data...)}
	}

	return f.Encode(toEncode, output)
}

func borsDecode(sc *ShardsContainer, writer io.Writer) (err error) {
	var shares []infectious.Share
	for ix, sh := range sc.shards {
		if sh != nil {
			shares = append(shares, infectious.Share{Number: ix, Data: sh.Data})
		}
	}
	f, err := infectious.NewFEC(sc.NumRequired, sc.NumTotal)
	if err != nil {return}

	fileContainerBytes, err := f.Decode(nil, shares)
	// TODO: Reed Solomon can correct. Perhaps better with more than the required number (?)
	if err != nil {return}

	r, err := sc.GetOriginalDataFromContainer(fileContainerBytes)
	if err != nil {return}

	_, err = io.Copy(writer, r)
	return
}
