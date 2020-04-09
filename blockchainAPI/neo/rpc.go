package neo

import (
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/interfaces"
	"github.com/joeqian10/neo-gogogo/helper"
	"github.com/joeqian10/neo-gogogo/nep5"
	"github.com/joeqian10/neo-gogogo/rpc"
	"github.com/joeqian10/neo-gogogo/rpc/models"
	"github.com/joeqian10/neo-gogogo/sc"
	"github.com/joeqian10/neo-gogogo/wallet"
	"github.com/pkg/errors"
	"strconv"
	"sync"
	"time"
)

// string addresses are base58 encoded public addresses (like AUePs1dfy5SLs65KRYRCozfk2SxZo3dFw3)
// aiming for 0.45c / GByte for upload = 0.00225 Gas / GByte
// There is no limit for a transactions size. It is free for tx under 1024 bytes.
// But when tx size is over 1024 bytes you will have to pay network fee for it. Fee = tx size * 0.00001 + 0.0001.

const (
	// These map names have to match exactly the names of the maps in the Neo contract
	spProfilesMap = "addressToProfile"	// SP
	nodeIdToAddrMap = "nodeIdToAddr"	// SP
	addressToUserNameMap = "addressToUserName"	// Uploader
)

var client *rpc.RpcClient
var neoEndpointOnce sync.Once
var neoEndpoint string

func RpcUrls() []string {
	// Defaults
	return []string{"http://seed3.ngd.network", "http://seed1.ngd.network"}
}

func NeoEndpoint() string {
	neoEndpointOnce.Do(func() {
		if neoEndpoint == "" {
			SetRpcUrl(RpcUrls())
		}
	})
	return neoEndpoint
}

func Client() *rpc.RpcClient {
	return rpc.NewClient(NeoEndpoint())
}

func SetRpcUrl(rpcUrls []string) {
	var port int
	switch BuildConfig {
	case Release:	port = 10333
	case Beta:		port = 20332
	case Debug:	neoEndpoint = "http://127.0.0.1:10002"; return
	default:		port = 0
	}
	neoEndpoint = FirstLiveUrl(rpcUrls, port)
}

func ArchonContractVersion() string {
	r, _, err := CallArchonContract("version", nil, nil, true)
	if err == nil {
		s, err := StringFromResponse(r)
		if err == nil {
			return s
		}
	}
	return err.Error()
}

func IsSpRegistered(address string) (isReg bool) {
	resp, err := GetStorageValueFromAddress(spProfilesMap, address)
	return err == nil && resp != ""
}

func UnregisterSp(sp *wallet.Account) (err error) {
	nodeId, err := GetNodeId(sp.Address)
	if err != nil {return}
	_, _, err = CallArchonContract(
		"unregisterStorageProvider",
		[]sc.ContractParameter{*addressParam(sp.Address),*stringParam(nodeId)},
		sp,
		true,
		)
	return
}

func RegisterSp(sp *wallet.Account, prof *NeoSpProfile) (txId string, err error) {
	// Costs about 3.8 Gas
	_, txId, err = CallArchonContract(
		"registerStorageProvider",
		[]sc.ContractParameter{
			*addressParam(sp.Address),
			*stringParam(prof.NodeId),
			*stringParam(prof.String()),
		},
		sp,
		true,
	)
	return
}

func GetSpProfile(address string) (*NeoSpProfile, error) {
	resp, err := GetStorageValueFromAddress(spProfilesMap, address)
	if err != nil {return nil, err}
	profS := HexStringToString(resp)
	prof, err := NewNeoSpProfile(profS)
	if err != nil {return nil, err}
	return prof, nil
}

func GetNodeId(address string) (nodeId string, err error) {
	prof, err := GetSpProfile(address)
	if err != nil {return}
	nodeId = prof.NodeId
	return
}

func GetSpMinAsk(address string) (minAsk int64, err error) {
	// Useful for debugging only, otherwise GetSpProfile is faster
	r, _, err := CallArchonContract(
		"getStorageProviderMinAsk",
		[]sc.ContractParameter{
			*addressParam(address),
		},
		nil,
		true,
	)
	if err == nil {
		minAsk, err = intFromResponseLog(r)
	}
	return
}

func GetSpAddress(nodeId string) (address string, err error) {
	address, err = GetAddressStorageValue(nodeIdToAddrMap, nodeId)
	return
}

func RegisterUserName(user *wallet.Account, userName string) (err error) {
	// This costs 1.6 Gas
	r, _, err := CallArchonContract(
		"registerUserName",
		[]sc.ContractParameter{
			*addressParam(user.Address),
			*stringParam(userName),
		},
		user,
		true,
	)
	if err == nil {
		ret, err := intFromResponseLog(r)
		if err == nil {
			if ret != 0 {
				err = fmt.Errorf("registration failed (code %d)", ret)
			}
		}
	}
	return
}

func UnregisterUserName(user *wallet.Account) (err error) {
	r, _, err := CallArchonContract(
		"unregisterUserName",
		[]sc.ContractParameter{
			*addressParam(user.Address),
		},
		user,
		true,
	)
	if err == nil {
		ret, err := intFromResponseLog(r)
		if err == nil {
			if ret != 0 {
				err = fmt.Errorf("unregister failed (code %d)", ret)
			}
		}
	}
	return
}

// GetUserName return the user name. Unregistered user returns empty string
func GetUserName(address string) (userName string, err error) {
	s, err := GetStorageValueFromAddress(addressToUserNameMap, address)
	if s != "" {
		userName = HexStringToString(s)
	}
	return
}

// ProposeUpload returns transaction ID if successful
func ProposeUpload(wallet *wallet.Account, pars *UploadParamsForNeo, payment int64, waitForConfirm bool) (txId string, err error) {
	// Consumes about 7 GAS
	//fmt.Println(pars)
	contractPars := []sc.ContractParameter{
		*addressParam(wallet.Address),
		*stringParam(I64ToA(payment)),	// could not make int param work
		*byteArrayParam(pars.Bytes()),
	}
	for _, spa := range pars.SPsToUploadTo {
		contractPars = append(contractPars,*addressParam(spa))
	}

	_, txId, err = CallArchonContract(
		"proposeUpload",
		contractPars,
		wallet,
		waitForConfirm,
	)
	// For debugging only
	//ret, _ := intFromResponseLog(l)
	//fmt.Println("ret:", ret)
	return
}

func GetUploadTxInfo(txId string) (pInfo *interfaces.UploadTxInfo, err error) {
	err = WaitForTransaction(txId)
	if err != nil {return}
	_, notification, err := getTxResponse(txId, false)
	if err != nil {return}
	if notification == "" {
		err = errors.New("Notification is missing")
		return
	}
	neoPars, err := NewUploadParamsForNeoFromBytes([]byte(notification))
	if err != nil {return}

	iPars := interfaces.UploadTxInfo{}
	iPars.TxId = txId
	iPars.UserName = neoPars.UserName
	iPars.PublicKey = StringToBytes(neoPars.PublicKey)
	iPars.FileContainerType = uint8(neoPars.FileContainerType)
	iPars.Signature = StringToBytes(neoPars.ContainerSignature)
	for _, spa := range neoPars.SPsToUploadTo {
		b, err2 := helper.AddressToScriptHash(spa)
		if err2 != nil {
			err = err2
			return
		}
		iPars.SPs = append(iPars.SPs, b.Bytes())
	}
	pInfo = &iPars
	return
}

func GetCGASBalanceOf(address string) (bal int64, err error) {
	n5h := nep5.NewNep5Helper(CgasScriptHash(), NeoEndpoint())
	addr, err := helper.AddressToScriptHash(address)
	if err != nil {return}
	balU, err := n5h.BalanceOf(addr)
	bal = int64(balU)
	return
}

func IsTxAccepted(txId string) (bool, error) {
	r1 := Client().GetRawTransaction(txId)
	if r1.HasError() {
		return false, errors.New("GetRawTransaction: " + r1.Error.Message)
	}
	return r1.Result.Confirmations > 0, nil
}

func WaitForTransaction(txId string) error {
	var checkPeriod, checkTimeout time.Duration
	fmt.Println("Waiting for transaction to complete...")
	switch BuildConfig {
	case Debug:
		checkPeriod = 2*time.Second
		checkTimeout = 8*time.Second
	default:
		checkPeriod = 9*time.Second
		checkTimeout = 45*time.Second
	}

	start := time.Now()
	first := true
	for time.Since(start) < checkTimeout {
		if first {
			time.Sleep(checkPeriod/2)
			first = false
		} else {
			time.Sleep(checkPeriod)
		}
		accepted, err := IsTxAccepted(txId)
		if err != nil {return err}
		if accepted {return nil}
	}
	return fmt.Errorf("timed out waiting for %s", txId)
}

func GetTxResponse(txId string, afterCall bool) (log *models.RpcApplicationLog, err error) {
	err = WaitForTransaction(txId)
	if err != nil {return}
	r, _, err2 := getTxResponse(txId, afterCall)
	if r != nil {
		log = &r.Result
	}
	if err2 != nil {
		err = err2
		return
	}
	return
}


func GetBlockHeight() (ret string, err error) {
	resp := Client().GetBlockCount()
	msg := resp.Error.Message
	if len(msg) != 0 {
		return "", fmt.Errorf("%s",msg)
	}
	// Need to return the previous block, so data from it as available
	return strconv.Itoa(resp.Result-1), nil
}

func GetBlockHash(height string) (hash string, err error) {
	index, err := strconv.Atoi(height)
	if err != nil {return}
	resp := Client().GetBlockHash(uint32(index))
	if resp.HasError() {
		err = errors.New(resp.Error.Message)
	} else {
		hash = resp.Result
	}
	return
}

func CgasString(gas int64) string {
	g := helper.Fixed8ToString(helper.NewFixed8(gas))
	return g + " CGAS"
}
