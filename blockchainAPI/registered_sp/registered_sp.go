package registeredsp

import (
	"github.com/pariz/gountries"
)

type RegisteredSp struct {
	IsInMarketPlace  bool
	Address          []byte
	SLALevel         int
	PledgedStorage   uint64
	RemainingStorage uint64
	Bandwidth        uint64
	CountryCode      gountries.Codes
	MinAskPrice      uint64
	NodeID           string
	Stake            uint64
	Url              string
}
