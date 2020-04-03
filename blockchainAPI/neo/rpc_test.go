package neo

import (
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/joeqian10/neo-gogogo/helper"
	"github.com/joeqian10/neo-gogogo/nep5"
	"github.com/joeqian10/neo-gogogo/sc"
	"github.com/joeqian10/neo-gogogo/tx"
	"gotest.tools/assert"

	"github.com/joeqian10/neo-gogogo/wallet"
	"log"
	"strings"
	"testing"
)

// Bootstrap on dev network
const bootStrapWif = "KwERwXSv5ctu1ne5yJxLDcAsCjiVgp3snX8pn8nBZUN3jDtFeaxH" // dev1.json
// address: 540c5c4180b09985968f22f80695349cfb7a945d
const dev1Wif = "KwERwXSv5ctu1ne5yJxLDcAsCjiVgp3snX8pn8nBZUN3jDtFeaxH"
const dev1Addr = "AQJgMwLnhJWj6RqvESSXyXwDsM3hxxfSEr"
const dev2Wif = "KydGZaqWyNrT9oN6oBistrmGfX8fzSoGqkAaNZreANeX9uc3PcSA"
// address: 8fcafb8b88199ec0622e2909383bf28187a06bfe Aey8DHNKSQCrHTpVRK32fLsJtXkAtdzSbJ
const dev3Wif = "L1X1kWnwzdYW8MQxLxzM1umCTfZmrCJUF8ZChKDEEbVR9WeYLZM9"
// address: 6a6663472c9d05ba1e98086d47138b47317609ce  AaZJBRcJ7DriYcy9ahJ38ZpZaPkgppLSU5
const sp1Wif = "L5mgtRVVZ2ikYgCYRNGxSXBNCMd3oKDWNbmKH3azAq5UMwjAZr3g"
const sp1Addr = "AaCkxSP1gkukyU5ZJPE4RiZC4mMtsjexnx" // 32d82775b3d8af02b17b4c319d384ae7fa1327ca
// NodeID on beta: QmY9EWeuE4yL4ccvwLtX9PP8baWcWE4Kw4fkGh7ZhsTvkc
const aWif = "AQzRMe3zyGS8W177xLJfewRRQZY2kddMun"

// CGAS asset ID = 0x602c79718b16e442de58778e148d0b1084e3b2dffd5de6b7b16cee7969282de7
// NEO asset ID = 0xc56f33fc6ecfcd0c225c4ab356fee59390af8560be0e930faebe74a6daff7c9b

func bootStrapProfile() *NeoSpProfile {
	prof := new(NeoSpProfile)
	prof.NodeId = BootStrapNodeId
	prof.CountryA3 = "USA"
	prof.PledgedStorage = 100
	prof.MinAsk = 200
	return prof
}

func bootStrapAccount() *wallet.Account {
	b, _ := wallet.NewAccountFromWIF(bootStrapWif)
	return b
}

func dev2Profile() *NeoSpProfile {
	// To be used on Debug only
	prof := new(NeoSpProfile)
	prof.NodeId = "NodeId"
	prof.CountryA3 = "USA"
	prof.PledgedStorage = 100
	prof.MinAsk = 200
	return prof
}

func a1Account() *wallet.Account {
	// For private net only
	b, _ := wallet.NewAccountFromWIF(aWif)
	return b
}

func dev1Account() *wallet.Account {
	b, _ := wallet.NewAccountFromWIF(dev1Wif)
	return b
}

func dev2Account() *wallet.Account {
	b, _ := wallet.NewAccountFromWIF(dev2Wif)
	return b
}

func dev3Account() *wallet.Account {
	b, _ := wallet.NewAccountFromWIF(dev3Wif)
	return b
}

func sp1Profile() *NeoSpProfile {
	// To be used on Debug only
	prof := new(NeoSpProfile)
	prof.NodeId = "QmY9EWeuE4yL4ccvwLtX9PP8baWcWE4Kw4fkGh7ZhsTvkc"
	prof.CountryA3 = "USA"
	prof.PledgedStorage = 100
	prof.MinAsk = 200
	return prof
}

func sp1Account() *wallet.Account {
	b, _ := wallet.NewAccountFromWIF(sp1Wif)
	return b
}

func uint160Address(addr string) helper.UInt160 {
	ua, _ := helper.AddressToScriptHash(addr)
	return ua
}

func cgasFromFloat(f float64) helper.Fixed8 {
	return helper.Fixed8FromFloat64(f)
}

func printCgas(c uint64) {
	c8 := helper.NewFixed8(int64(c))
	cf := (helper.Fixed8ToFloat64(c8))
	log.Println(cf)
}

func cGasBalanceFromAddress(addrS string, t *testing.T) {
	n5h := nep5.NewNep5Helper(cgasScriptHash(), NeoEndpoint)
	addr, err := helper.AddressToScriptHash(addrS)
	if err != nil {log.Fatalln(err)}
	bal, err := n5h.BalanceOf(addr)
	if err != nil {log.Fatalln(err)}
	printCgas(bal)
}

func cGasBalance(acc *wallet.Account, t *testing.T) {
	cGasBalanceFromAddress(acc.Address, t)
}

func printCGASBalances(from, to helper.UInt160) {
	nep5Helper := nep5.NewNep5Helper(cgasScriptHash(), NeoEndpoint)
	bal1, _ := nep5Helper.BalanceOf(from)
	bal2, _ := nep5Helper.BalanceOf(to)
	log.Println(bal1,bal2)
}

func printCGASBalancesFromAddress(from, to string) {
	nep5Helper := nep5.NewNep5Helper(cgasScriptHash(), NeoEndpoint)
	f, _ := helper.AddressToScriptHash(from)
	bal1, _ := nep5Helper.BalanceOf(f)
	t, _ := helper.AddressToScriptHash(to)
	bal2, _ := nep5Helper.BalanceOf(t)
	log.Println(bal1,bal2)
}

func TestRpcCall1(t *testing.T) {
	u, _ := wallet.NewAccountFromWIF("KzTkuYexzCxdvDKRys2mn4wSBqoXbHm7xJHdVA3t6dMyQdtHqVUa")
	pars := new(UploadParamsForNeo)
	pars.UserName="marius"
	pars.FileContainerType=1
	pars.ContainerSignature = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	pars.SPsToUploadTo = []string{"AWdcDR2dhs3tFCHGzjdJ6vtVZybVF3c7eE"}
	txId, err := ProposeUpload(u,pars,200, true)
	if err != nil {t.Fatal(err)}
	info, err := GetUploadTxInfo(txId)
	if err != nil {t.Fatal(err)}
	t.Log(info.ToJsonString())

	//v := ArchonContractVersion()
	//t.Log(v)
}

func TestIsSpRegistered(t *testing.T) {
	b := bootStrapAccount()
	ok := IsSpRegistered(b.Address)
	if !ok {
		log.Fatalf("not registered")
	}
}

func TestArchonContractVersion(t *testing.T) {
	v := ArchonContractVersion()
	if !strings.HasPrefix(v, "Archon") {
		log.Fatalf("Wrong Archon version: %q", v)
	}
}

func logAddressBytes(acc *wallet.Account) {
	ah, _ := helper.AddressToScriptHash(acc.Address)
	abs := ah.String()
	log.Println(abs)
}

func testReg(t *testing.T, acc *wallet.Account, prof *NeoSpProfile) {
	if !IsSpRegistered(acc.Address) {
		txId, err := RegisterSp(acc, prof)
		if err != nil {log.Fatal(err)}
		log.Println("Reg tx id=", txId)
	}
	logAddressBytes(acc)

	addr, err := GetSpAddress(prof.NodeId)
	if err != nil {log.Fatal(err)}
	assert.Equal(t, acc.Address,addr)
	assert.Equal(t, true, IsSpRegistered(acc.Address))
}

func TestRegisterBootstrap(t *testing.T) {
	log.Println("Register bootstrap")
	testReg(t, bootStrapAccount(), bootStrapProfile())
}

func TestUnregisterBootstrap(t *testing.T) {
	bootstrap := bootStrapAccount()
	err := UnregisterSp(bootstrap,BootStrapNodeId)
	if err != nil {log.Fatal(err)}
}

func TestUnregisterSp(t *testing.T) {
	acc := sp1Account()
	prof := sp1Profile()
	err := UnregisterSp(acc,prof.NodeId)
	if err != nil {log.Fatal(err)}
}

func TestRegisterSp(t *testing.T) {
	testReg(t, sp1Account(), sp1Profile())
}

func TestGetSpProfile(t *testing.T) {
	sp1 := sp1Account()
	profRet, err := GetSpProfile(sp1.Address)
	if err != nil {log.Fatal(err)}
	assert.Equal(t, sp1Profile().MinAsk, profRet.MinAsk)
	log.Println(profRet)
	minAsk, err := GetSpMinAsk(sp1.Address)
	if err != nil {log.Fatal(err)}
	assert.Equal(t, profRet.MinAsk, minAsk)
}

func registerUserName(acc *wallet.Account, name string, t *testing.T) {
	n, err := GetUserName(acc.Address)
	if err != nil {log.Fatal(err)}
	if n == "" {
		err := RegisterUserName(acc, name)
		if err != nil {
			log.Fatal(err)
		}
		n, err = GetUserName(acc.Address)
		if err != nil {log.Fatal(err)}
		assert.Equal(t,name, n)
	}
	log.Println("User name is", n)
}

func TestRegisterUserName(t *testing.T) {
	registerUserName(dev1Account(),"dev1", t)
}

func TestUnregisterUserName(t *testing.T) {
	boot := dev3Account()
	err := UnregisterUserName(boot)
	if err != nil {log.Fatal(err)}
	n, err := GetUserName(boot.Address)
	if err != nil {log.Fatal(err)}
	assert.Equal(t,"", n)
}

func TestCGasName(t *testing.T) {
	cgas := nep5.NewNep5Helper(cgasScriptHash(), NeoEndpoint)
	s, err := cgas.Name()
	if err != nil {log.Fatal(err)}
	assert.Equal(t,"NEP5 GAS", s)
}

func TestGetCGASNameFromArchon(t *testing.T) {
	l, _, err := callArchonContract(
		"getCgasName",
		[]sc.ContractParameter{},
		nil,
		true,
	)
	if err != nil {log.Fatal(err)}
	s, err := stringFromResponse(l)
	if err != nil {log.Fatal(err)}
	log.Println(s)
}

func TestReverseBytes(t *testing.T) {
	s := "0x76db3192722022eb7841038246dc8fa636dcf274"
	b := StringToBytes(s)
	b = helper.ReverseBytes(b)
	s = BytesToString(b)
	log.Println(s)
}

func TestDecodeNeoString(t *testing.T) {
	//s := hexStringToString("0x1f8b08000000000000ff14cc3b6a9c410c00e02be9392395614913082c096eb6d38c24632f76e307bfe13fbcf105beac4f3c")
	//s := hexStringToString("0xf74696669636174696f6e")
	s := hexStringToString("0x3230307c3130307c5553417c516d5939455765754534794c34636376774c745839505038626157635745344b7734666b4768")
	log.Println(s)
}

func TestGetAccountState(t *testing.T) {
	acc := sp1Account()
	resp := Client().GetAccountState(acc.Address)
	log.Println(resp.ErrorResponse.HasError())
	r1 := Client().GetUnspents(acc.Address)
	log.Println(r1.ErrorResponse.HasError())
	tb := tx.NewTransactionBuilder(NeoEndpoint)
	f, _ := helper.AddressToScriptHash(acc.Address)
	inputs, totalPay, err := tb.GetTransactionInputs(f, tx.GasToken, helper.Fixed8FromFloat64(0.1))
	if err != nil {log.Fatalln(err)}
	log.Println(totalPay,inputs)
}

func TestGetUploadTxInfo(t *testing.T) {
	info, err := GetUploadTxInfo("577d3ee9d9b73a61439cf121fdc7998a7e3a0275d9c2612479bba1181c1259e4")
	if err != nil {log.Fatalln(err)}
	log.Println(info.ToJsonString())
}

func TestGetBlockHash(t *testing.T) {
	height, err := GetBlockHeight()
	if err != nil {log.Fatalln(err)}
	hash, err := GetBlockHash(height)
	if err != nil {log.Fatalln(err)}
	log.Println(hash)
}

func TestCgasTotalSupply(t *testing.T) {
	n5h := nep5.NewNep5Helper(cgasScriptHash(), NeoEndpoint)
	total, err := n5h.TotalSupply()
	if err != nil {log.Fatalln(err)}
	printCgas(total)
}

func TestSp1AndDev1(t *testing.T) {
	testReg(t, sp1Account(), sp1Profile())
	registerUserName(dev1Account(),"dev1", t)
}

func TestSendNEO(t *testing.T) {
	// Private net only
	a := a1Account()
	Client().SendFrom(tx.NeoTokenId, a.Address, a.Address, 100000000, 0.0, a.Address )
}

func TestMintCGas(t *testing.T) {
	cgas := (*nep5.CgasHelper)(nep5.NewNep5Helper(cgasScriptHash(), NeoEndpoint))
	txId, err := cgas.MintTokens(dev1Account(),10.0)
	if err != nil {log.Fatal(err)}
	log.Println(txId)
	txId, err = cgas.MintTokens(sp1Account(),10.0)
	if err != nil {log.Fatal(err)}
	log.Println(txId)
}

func TestProposeUpload(t *testing.T) {
	uplAcc := dev1Account()
	spAcc := sp1Account()
	printCGASBalancesFromAddress(uplAcc.Address,spAcc.Address)
	pars := UploadParamsForNeo{
		UserName:           "dev1",
		PublicKey:          "9fc12039631a920a8106237c635a82c7211f8fe1a2869a75aec0c1f89f5f2420c63f9f0a706cf697e9fe2496e41006abe29faee8738c5391bf6224eece434262",
		ContainerSignature: "0x85b551a45f5f21ed93ecd18ae425578a2ced9ddd6dfa594023371fd4e17fb3ed18756b31aeed4dd4502a20350f08dfb3b4d3a9d967ea8aff84edabe059da103000",
		FileContainerType:  0,
		SPsToUploadTo:      []string{spAcc.Address},
	}
	txId, err := ProposeUpload(uplAcc, &pars, 210, true)
	if err != nil {log.Fatalln(err)}
	log.Println("txId", txId)
	printCGASBalancesFromAddress(uplAcc.Address,spAcc.Address)
}

func TestTransferCGAS(t *testing.T)  {
	fromAccount := dev1Account()
	from, err := helper.AddressToScriptHash(fromAccount.Address)
	if err != nil {log.Fatalln(err)}
	to, err := helper.AddressToScriptHash(sp1Addr)
	if err != nil {log.Fatalln(err)}
	printCGASBalances(from, to)

	amount := helper.NewFixed8(100)
	cp1 := sc.ContractParameter{
		Type:  sc.Hash160,
		Value: from.Bytes(),
	}
	cp2 := sc.ContractParameter{
		Type:  sc.Hash160,
		Value: to.Bytes(),
	}
	cp3 := sc.ContractParameter{
		Type:  sc.Integer,
		Value: amount.Value,
	}
	_, txId, err := callArchonContract("transferCGAS", []sc.ContractParameter{cp1,cp2,cp3}, fromAccount, true)
	log.Println("txId", txId)
	if err != nil {log.Fatalln(err)}

	printCGASBalances(from, to)
}
