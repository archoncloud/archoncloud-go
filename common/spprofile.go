package common

import (
	"fmt"
	ecommon "github.com/ethereum/go-ethereum/common"
	"sort"
	"strings"
)

// Urls for SP. Can have http, https or both, but at least one
type Urls struct {
	Host      string `json:"Host"`
	HttpPort  string `json:"http"`
	HttpsPort string `json:"https"`
}

const UrlsSep = "|"

func NewUrls(enc string) (*Urls, error) {
	u := Urls{}
	a := strings.Split(enc, UrlsSep)
	if len(a) != 3 {
		return nil, fmt.Errorf("NewUrls needs 3 fields. Urls=%q", enc)
	}
	u.Host = a[0]
	u.HttpPort = a[1]
	u.HttpsPort = a[2]
	return &u, nil
}

func (u *Urls) String() string {
	return u.Host + UrlsSep + u.HttpPort + UrlsSep + u.HttpsPort
}

func (u *Urls) Url(preferHttp bool) string {
	useHttp := true
	if preferHttp {
		useHttp = u.HttpPort != ""
	} else {
		useHttp = u.HttpsPort == ""
	}
	if useHttp {
		return "http://"+u.Host +":"+u.HttpPort
	}
	return "https://"+u.Host +":"+u.HttpsPort
}

type SpProfile struct {
	Urls		Urls `json:"urls"`
	// The following for permissioned only
	Address		string `json:"address"`	// blockchain address
	// in the currency of the layer, per MByte per month
	// (for Eth: wei/Mbyte)
	MinAskPrice        int64   `json:"min_ask"`
	AvailableGigaBytes float64 `json:"available_giga_bytes"`
	PledgedGigaBytes	float64 `json:"pledged_giga_bytes"`
	NodeId             string  `json:"node_id"`
}

func (sp *SpProfile) Host() string {
	return sp.Urls.Host
}

func (sp *SpProfile) Url(preferHttp bool) string {
	return sp.Urls.Url(preferHttp)
}

func (sp *SpProfile) IsValid() bool {
	if sp.Urls.Host == "" {return false}
	if sp.Address == "" {return false}
	if sp.MinAskPrice < 0 {return false}
	return true
}

func (sp *SpProfile) AddressBytes() []byte {
	return StringToBytes(sp.Address)
}

type StorageProviders []SpProfile

func NewStorageProviders(num int) StorageProviders {
	sps := make([]SpProfile,num)
	return sps
}

func (sps *StorageProviders) ForAllProfiles(do func(*SpProfile)) {
	for _, sp := range *sps {
		do(&sp)
	}
}

func (sps *StorageProviders) KeepOnly(n int) {
	*sps = (*sps)[:n]
}

func (sps *StorageProviders) Num() int {
	return len(*sps)
}

func (sps *StorageProviders) Add(sp *SpProfile) {
	*sps = append(*sps, *sp)
}

func (sps *StorageProviders) Set(i int, sp *SpProfile) {
	(*sps)[i] = *sp
}

func (sps *StorageProviders) Get(i int) *SpProfile {
	return &(*sps)[i]
}

func (sps *StorageProviders) GetOfAddress(addr string) *SpProfile {
	for _, sp := range *sps {
		if sp.Address == addr {
			return &sp
		}
	}
	return nil
}

func (sps *StorageProviders) Addresses() (addr []string) {
	for _, sp := range (*sps) {
		addr = append(addr, sp.Address)
	}
	return
}

func (sps *StorageProviders) Hosts() (hosts []string) {
	for _, sp := range *sps {
		hosts = append(hosts, sp.Urls.Host)
	}
	return
}

func (sps *StorageProviders) EthAddresses() [][ecommon.AddressLength]byte{
	spa := make([][ecommon.AddressLength]byte, sps.Num())
	for ix, sp := range *sps {
		copy(spa[ix][:], StringToBytes(sp.Address))
	}
	return spa
}

// Sort by min ask price
func (sps *StorageProviders) SortByMinAsk(granularity uint64) {
	sort.Slice(*sps, func(i, j int) bool {
		return RoundUp(uint64((*sps)[i].MinAskPrice),granularity) < RoundUp(uint64((*sps)[j].MinAskPrice), granularity)
	})
}

// pickRandom returns up to needed random SPs from input sps
func (sps *StorageProviders) PickRandom(needed int) StorageProviders {
	if sps.Num() < needed {
		return *sps
	}
	rr := RandomRange(needed, 0, sps.Num())
	randomSps := NewStorageProviders(needed)
	for i := 0; i < needed; i++ {
		randomSps[i] = (*sps)[rr[i]]
	}
	return randomSps
}

// in currency of MinAskPrice, which depends on the blockchain
// for Eth it is Wei, for Neo it is Gas
func (sps *StorageProviders) PriceOfUpload(numBytes int64) int64 {
	var minAskPerMByte []int64
	var highest int64 = 0
	for _, sp := range *sps {
		a := sp.MinAskPrice
		minAskPerMByte = append(minAskPerMByte, a)
		if a > highest {
			highest = a
		}
	}
	mBytes := DivideRoundUp(uint64(numBytes),Mega)
	// TODO: multiply by months, once it is part of registration
	total := highest * int64(mBytes) * int64(sps.Num())

/*	fmt.Println( "min ask per MByte", minAskPerMByte)
	fmt.Println("highest", highest)
	fmt.Println("num bytes", numBytes)
	fmt.Println("num MBytes", mBytes)
	fmt.Println("total charged, Wei", total)
*/
	return total
}

func (sps *StorageProviders) Remove(removeThis func(profile *SpProfile) bool) {
	newA := StorageProviders{}
	for _, sp := range *sps {
		if !removeThis(&sp) {
			newA = append(newA, sp)
		}
	}
	*sps = newA;
}
