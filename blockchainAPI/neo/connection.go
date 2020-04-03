package neo

import (
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/joeqian10/neo-gogogo/helper"
)

// To deploy contract. For "public static object Main(string method, object[] args)"
// Parameter List: 0710
// Return Type: 10

func cgasScript() string {
	switch BuildConfig {
	case Debug:	return "0x2a8cc3b07d25dfae0d212738b39c2e0d17a1c7f3" 	// local computer
	case Beta:	return "0x74f2dc36a68fdc4682034178eb2220729231db76"		// testnet
	default: return ""
	}
}

func archonCloudScript() string {
	switch BuildConfig {
	case Debug:	return "0xdd8491f941dff98947bcc88f9786c4f7e2419ea2"	// local computer
	case Beta:	return "0xf1494e3987e0c4f35695cde582c9feb905845644"	// AWS testnet
	default:	return ""
	}
}

func cgasScriptHash()  (scriptHash helper.UInt160)  {
	scriptHash, _ = helper.UInt160FromString(cgasScript())
	return
}
