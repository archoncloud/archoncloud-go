package neo

import (
	"encoding/hex"
	"errors"
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/joeqian10/neo-gogogo/helper"
	"github.com/joeqian10/neo-gogogo/rpc"
	"github.com/joeqian10/neo-gogogo/rpc/models"
	"github.com/joeqian10/neo-gogogo/sc"
	"github.com/joeqian10/neo-gogogo/tx"
	"github.com/joeqian10/neo-gogogo/wallet"
	"strconv"
	"strings"
)

const (
	stringSep = "|"	// Also used in contract
)

func archonCloudScriptHash() (scriptHash helper.UInt160) {
	scriptHash, err := helper.UInt160FromString(archonCloudScript())
	if err != nil {
		panic(err)
	}
	return
}

func CgasScriptHash()  (scriptHash helper.UInt160)  {
	scriptHash, _ = helper.UInt160FromString(cgasScript())
	return
}

// CallArchonContract calls a method in the Archon Cloud contract. It will be signed if From is not nil.
func CallArchonContract(method string, args []sc.ContractParameter, from *wallet.Account, waitForConfirm bool) (log *models.RpcApplicationLog, txId string, err error) {
	sb := sc.NewScriptBuilder()
	sb.MakeInvocationScript(archonCloudScriptHash().Bytes(), method, args)
	script := sb.ToArray()
	tb := tx.NewTransactionBuilder(NeoEndpoint())
	gas, err := tb.GetGasConsumed(script)
	if err != nil {return}
	if gas.GreaterThan(helper.Zero) {
		err = fmt.Errorf("GAS consumed exceeds free tier")
		return
	}

	myTx := tx.NewInvocationTransaction(script)
	if from != nil {
		err = tx.AddSignature(myTx, from.KeyPair)
		if err != nil {return}
	}
	response := Client().SendRawTransaction(myTx.RawTransactionString())
	if response.HasError() {
		err = errors.New(response.Error.Message)
		return
	}
	txId = myTx.HashString()
	if waitForConfirm {
		log, err = GetTxResponse(txId, true)
	}
	return
}

func Account(wif string) *wallet.Account {
	acc, _ := wallet.NewAccountFromWIF(wif)
	return acc
}

func addressParam(address string) *sc.ContractParameter {
	addr, _ := helper.AddressToScriptHash(address)
	param := sc.ContractParameter{
		Type:  sc.Hash160,
		Value: addr.Bytes(),
	}
	return &param
}

func intParam(v int64) *sc.ContractParameter {
	param := sc.ContractParameter{
		Type:  sc.Integer,
		Value: v,
	}
	return &param
}

func byteArrayParam(b []byte) *sc.ContractParameter {
	param := sc.ContractParameter{
		Type:  sc.ByteArray,
		Value: b,
	}
	return &param
}

func addressArrayParam(a [][]byte) *sc.ContractParameter {
	var pa []sc.ContractParameter
	for _, addr := range a {
		pa = append(pa,sc.ContractParameter{
			Type: sc.ByteArray,
			Value: addr,
		})
	}
	param := sc.ContractParameter{
		Type:  sc.Array,
		Value: pa,
	}
	return &param
}

func stringParam(v string) *sc.ContractParameter {
	// TODO: Need to find out all the other limitations
	if len(v) > 1024 {
		panic("Neo string parameter exceeds 1024")
	}
	param := sc.ContractParameter{
		Type:  sc.String,
		Value: v,
	}
	return &param
}

func processResponse(log *models.RpcApplicationLog, f func(typ, val string) error) (err error) {
	if log != nil && len(log.Executions) > 0 {
		e0 := log.Executions[0]
		if len(e0.Stack) > 0 {
			item := e0.Stack[len(e0.Stack)-1]
			err = f(item.Type, item.Value.(string))
			return
		}
	}
	err = errors.New("there was no return value")
	return
}

func checkForErrorInResponse(log *models.RpcApplicationLog) (err error) {
	if log != nil && len(log.Executions) > 0 {
		e0 := log.Executions[0]
		if len(e0.Stack) > 0 {
			top := e0.Stack[len(e0.Stack)-1]
			if top.Type == "ByteArray" {
				s := HexStringToString(top.Value.(string))
				if strings.HasPrefix(s, "Error") {
					err = fmt.Errorf(s)
				}
			}
		}
	}
	return
}

/*
func boolFromResponse(r *rpc.GetApplicationLogResponse) (result bool, err error) {
	err = processResponse(r, func(typ, val string) error {
		var err2 error
		switch typ {
		case "Boolean":
			result, err2 = strconv.ParseBool(val)
			return err2
		case "Integer":
			i, err2 := strconv.Atoi(val)
			if err2 == nil {
				result = i != 0
			}
			return err2
		}
		return errors.New("returned value is not Boolean")
	})
	return
}
*/

func intFromResponseLog(r *models.RpcApplicationLog) (i int64, err error) {
	err = processResponse(r, func(typ, val string) error {
		var err2 error
		switch typ {
		case "Integer":
			i, err2 = strconv.ParseInt(val, 10, 64)
			return err2
		}
		return errors.New("returned value is not Integer")
	})
	return
}

func StringFromResponse(log *models.RpcApplicationLog) (s string, err error) {
	err = processResponse(log, func(typ, val string) error {
		switch typ {
		case "ByteArray":
			s = HexStringToString(val)
			return nil
		}
		return errors.New("returned value is not a string")
	})
	return
}

func byteSliceFromResponse(log *models.RpcApplicationLog) (b []byte, err error) {
	err = processResponse(log, func(typ, val string) error {
		switch typ {
		case "ByteArray":
			b = stringToBytes(val)
			return nil
		}
		return errors.New("returned value is not a byte array")
	})
	return
}

func byteSliceStringFromResponse(log *models.RpcApplicationLog) (s string, err error) {
	err = processResponse(log, func(typ, val string) error {
		switch typ {
		case "ByteArray":
			s = val
			return nil
		}
		return errors.New("returned value is not a byte array")
	})
	return
}

func stringToBytes(input string) (b []byte) {
	if len(input) == 0 {return}

	if strings.HasPrefix(input, "0x") {
		input = input[2:]
	}
	b, _ = hex.DecodeString(input)
	return
}

func HexStringToString(s string) string {
	b := stringToBytes(s)
	return string(b)
}

// printNotifications is for debugging only
//func printNotifications(r *rpc.GetApplicationLogResponse) {
//	if len(r.Result.Executions) != 0 {
//		e := r.Result.Executions[0]
//		for _, n := range e.Notifications {
//			if n.State.Type == "Array" {
//				for _, v := range n.State.Value.([]interface{}) {
//					for _, value := range v.(map[string]interface{}) {
//						fmt.Println(HexStringToString(value.(string)))
//					}
//				}
//			}
//		}
//	}
//}

func getTxResponse(txId string, afterCall bool) (log *rpc.GetApplicationLogResponse, notification string, err error) {
	r:= Client().GetApplicationLog(txId)
	if r.HasError() {
		err = errors.New(r.Error.Message)
		return
	}
	if len(r.Result.Executions) == 0 {
		err = errors.New("no executions")
		return
	}
	e := r.Result.Executions[0]
	if afterCall {
		// for debugging only
		LogDebug.Println("Gas consumed:", e.GasConsumed)
	}
	if e.VMState == "FAULT" {
		err = errors.New("Neo VM Fault")
		return
	}
	log = &r
	for _, n := range e.Notifications {
		if n.State.Type == "Array" {
			values := n.State.Value
			if len(values) == 2 {
				v := getValue(&values[1])
				switch getValue(&values[0]) {
				case "notification":
					notification = v
					return
				case "error":
					err = errors.New(v)
					return
				}
			}
		}
	}
	return
}

func getValue(par *models.RpcContractParameter) string {
	if par != nil {
		if par.Type == "ByteArray" {
			return HexStringToString(par.Value.(string))
		}
	}
	return ""
}

//func createWitnesses(from *wallet.Account, t *tx.InvocationTransaction) (ws []*tx.Witness) {
//	additionalSignature, err := from.KeyPair.Sign(t.UnsignedRawTransaction())
//	if err != nil {return}
//	sb2 := sc.NewScriptBuilder()
//	_ = sb2.EmitPushBytes(additionalSignature)
//	additionalWitness, err := tx.CreateWitness(sb2.ToArray(), keys.CreateSignatureRedeemScript(from.KeyPair.PublicKey))
//	if err != nil {return}
//	ws = tx.WitnessSlice{additionalWitness}
//	return
//}

func AddressToBytes(address string) []byte {
	addr, _ :=  helper.AddressToScriptHash(address)
	return addr.Bytes()
}

func AddressToHexString(address string) string {
	b := AddressToBytes(address)
	//b = helper.ReverseBytes(b)
	return strings.TrimPrefix(BytesToString(b),"0x")
}

func AddressFromHexString(address string) string {
	buf := StringToBytes(address)
	u, err := helper.UInt160FromBytes(buf)
	if err != nil {return ""}
	return helper.ScriptHashToAddress(u)
}

func AddressBytesFromHexString(address string) []byte{
	return AddressToBytes(AddressFromHexString(address))
}

func GetStorageValue(table, key string) (string, error) {
	// Make the key
	b := []byte{}
	b = append(b, []byte(table)...)
	b = append(b, 0)
	b = append(b, []byte(key)...)
	encodedKey := strings.TrimPrefix(BytesToString(b), "0x")
	resp := Client().GetStorage(archonCloudScript(), encodedKey)
	if resp.Error.Message != "" {return "", errors.New(resp.Error.Message)}
	return resp.Result, nil
}

func GetStorageValueFromBytesKey(table string, key []byte) (string, error) {
	// Make the key
	b := []byte{}
	b = append(b, []byte(table)...)
	b = append(b, 0)
	b = append(b, key...)
	encodedKey := strings.TrimPrefix(BytesToString(b), "0x")
	resp := Client().GetStorage(archonCloudScript(), encodedKey)
	if resp.HasError() {return "", errors.New(resp.Error.Message)}
	return resp.Result, nil
}

func GetStringStorageValue(table, key string) (string, error) {
	s, err := GetStorageValue(table, key)
	if err != nil {return "", err}
	return HexStringToString(s), nil
}

func GetStorageValueFromAddress(table, address string) (value string, err error) {
	value, err = GetStorageValueFromBytesKey(table, AddressToBytes(address))
	return
}

func GetAddressStorageValue(table, key string) (addr string, err error) {
	s, err := GetStorageValue(table, key)
	if err != nil {return}
	b, err := hex.DecodeString(s)
	if err != nil {return}
	u, err := helper.UInt160FromBytes(b)
	if err != nil {return}
	addr = helper.ScriptHashToAddress(u)
	return
}
