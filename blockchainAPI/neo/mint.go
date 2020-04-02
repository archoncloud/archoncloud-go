package neo

import (
	"fmt"
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/joeqian10/neo-gogogo/helper"
	"github.com/joeqian10/neo-gogogo/nep5"
	"github.com/joeqian10/neo-gogogo/wallet"
	"os"
	"strconv"
)

// MintCGasIfNeeded will exit/abort if not enough available
func MintCGasIfNeeded(acc *wallet.Account, cgas int64) {
	n5h := nep5.NewNep5Helper(cgasScriptHash(),neoEndpoint())
	addr, err := helper.AddressToScriptHash(acc.Address)
	Abort(err)
	bal, err := n5h.BalanceOf(addr)
	Abort(err)
	if bal >= uint64(cgas) {
		// All is good
		return
	}
	msg := fmt.Sprintf("Your account has %s. %s needed. Do you want to mint CGAS",
		CgasString(int64(bal)), CgasString(cgas))
	if !Yes(msg) {os.Exit(0)}
	toMint := PromptForInput("How much CGAS do you want minted? Enter amount:")
	if toMint == "" {os.Exit(0)}
	toMintAmount, err := strconv.ParseFloat(toMint,64)
	Abort(err)
	cgh := (*nep5.CgasHelper)(n5h)
	txId, err := cgh.MintTokens(acc,toMintAmount)
	Abort(err)
	fmt.Println("Mint Tx Id=", txId)
	return
}
