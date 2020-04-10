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
	case Debug:	return "0x14c2f62a9c27cebe9d384d1bc0d290f59d305caf"	// local computer
	case Beta:	return "0x0c2f70c7f843a2f2901b9d1ed3059401109b74ca"	// testnet
	default:	return ""
	}
}
