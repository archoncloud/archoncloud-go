package neo

import (
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"strconv"
	"strings"
)

type NeoSpProfile struct {
	MinAsk         int64	// Gas per MByte
	PledgedStorage int64	// bytes
	CountryA3      string
	NodeId         string
}

func (p *NeoSpProfile) String() string {
	// Contract assumes MinAsk to be first item
	return SeparatedStringList(stringSep,
			p.MinAsk,
			p.PledgedStorage,
			p.CountryA3,
			p.NodeId,
		)
}

func NewNeoSpProfile(s string) (p *NeoSpProfile, err error) {
	a := strings.Split(s, stringSep)
	if len(a) != 4 {
		err = fmt.Errorf("invalid profile string: %q", s)
		return
	}
	p = new(NeoSpProfile)
	p.MinAsk, err = strconv.ParseInt(a[0], 10, 64)
	if err != nil {return}
	p.PledgedStorage, err = strconv.ParseInt(a[1], 10, 64)
	if err != nil {return}
	p.CountryA3 = a[2]
	p.NodeId = a[3]
	err = nil
	return
}
