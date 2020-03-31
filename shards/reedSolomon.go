package shards

import (
	. "github.com/itsmeknt/archoncloud-go/common"
	"github.com/vivint/infectious"
)

func GenerateRSIParity(input []byte, dataShards, parityShards int) ([][]byte, error) {
	parity := make([][]byte, parityShards)
	total := dataShards+parityShards
	f, err := infectious.NewFEC(dataShards, total)
	if err == nil {
		data := ExtendedSliceToMultipleOf(dataShards, input)
		output := func(s infectious.Share) {
			if s.Number >= dataShards {
				// The share does not persist, need to copy
				parity[s.Number-dataShards] = append([]byte(nil), s.Data...)
			}
		}
		err = f.Encode(data, output)
		if err != nil {
			return nil, err
		}
	}
	return parity, nil
}

func GenerateRSIParity_4_5(input []byte) ([]byte, error) {
	p, err := GenerateRSIParity(input, 4, 1)
	if err != nil {return nil, err}
	return  p[0], nil
}

func GenerateRSIParity_25_26(input []byte) ([]byte, error) {
	p, err := GenerateRSIParity(input, 25, 1)
	if err != nil {return nil, err}
	return  p[0], nil
}

// CorrectRSI takes input as data and some parity shards previously generated and returns the
// corrected data as output. Using infectious
func CorrectRSI(input []byte, parity [][]byte, dataShards, parityShards int) (output []byte, err error) {
	total := dataShards+parityShards
	f, err := infectious.NewFEC(dataShards, total)
	if err == nil {
		data := ExtendedSliceToMultipleOf(dataShards, input)
		slen := len(data) / dataShards
		shares := make([]infectious.Share,0)
		for i := 0; i < dataShards; i++ {
			shares = append(shares, infectious.Share{
				Number: i,
				Data:   data[i*slen:(i+1)*slen],
			})
		}
		for i := dataShards; i < total; i++ {
			shares = append(shares, infectious.Share{
				Number: i,
				Data:   parity[i],
			})
		}
		output, err = f.Decode(nil, shares)
		if err == nil {
			output = output[:len(input)]
		}
	}
	return
}

func GetReedSolomonShareSize(inputSize int64, required int) int64 {
	return (inputSize + int64(required) - 1) / int64(required)
}