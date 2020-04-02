package neo

import (
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/joeqian10/neo-gogogo/helper"
)

// To deploy contract. For "public static object Main(string method, object[] args)"
// Parameter List: 0710
// Return Type: 10

func neoEndpoint() string {
	switch BuildConfig {
	case Debug: return "http://127.0.0.1:10002"
	case Dev: return "http://13.57.196.239:10002"	// AWS private node
	case Beta:
		// testnet
		return FirstLiveUrl([]string{"http://seed1.ngd.network:20332","http://13.57.196.239:20332" })
	default: return ""
	}
}

func cgasScript() string {
	switch BuildConfig {
	case Debug, Dev:	return "0x2a8cc3b07d25dfae0d212738b39c2e0d17a1c7f3" 	// one-node
	case Beta:			return "0x74f2dc36a68fdc4682034178eb2220729231db76"		// testnet
	default: return ""
	}
}

func archonCloudScript() string {
	switch BuildConfig {
	case Debug:	return "0xdd8491f941dff98947bcc88f9786c4f7e2419ea2"	// local computer
	case Dev:	return "0xc6079f27508540c3450bd78283c25a15cca6b954"	// AWS private node
	case Beta:	return "0xf1494e3987e0c4f35695cde582c9feb905845644"	// AWS testnet
	default:	return ""
	}
}

func archonCloudScriptHash() (scriptHash helper.UInt160) {
	scriptHash, err := helper.UInt160FromString(archonCloudScript())
	if err != nil {
		panic(err)
	}
	return
}

func cgasScriptHash()  (scriptHash helper.UInt160)  {
	scriptHash, _ = helper.UInt160FromString(cgasScript())
	return
}
