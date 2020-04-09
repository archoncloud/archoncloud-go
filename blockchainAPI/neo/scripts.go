package neo

import (
	. "github.com/archoncloud/archoncloud-go/common"
)

func cgasScript() string {
	switch BuildConfig {
	case Debug:	return "0x2a8cc3b07d25dfae0d212738b39c2e0d17a1c7f3" 	// local computer
	case Beta:	return "0x74f2dc36a68fdc4682034178eb2220729231db76"		// testnet
	default: return ""
	}
}

func archonCloudScript() string {
	// To deploy contract. For "public static object Main(string method, object[] args)"
	// Parameter List: 0710
	// Return Type: 10
	switch BuildConfig {
	case Debug:	return "0x88aedb8c60fed5c565360fc823474c3fa91e81ac"	// local computer
	case Beta:	return "0xf1494e3987e0c4f35695cde582c9feb905845644"	// testnet
	default:	return ""
	}
}
