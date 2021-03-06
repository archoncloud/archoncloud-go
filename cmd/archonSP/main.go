// arconSP is the storage provider for the Archon CloudStorage
package main

import (
	. "github.com/archoncloud/archoncloud-go/common"
	"github.com/archoncloud/archoncloud-go/storageProvider"
)

func main() {
	// Initialize logging (rotating logger)
	InitLogging(storageProvider.GetLogFilePath())

	storageProvider.SetupAccountAndDht()

	// Start the web server. Will run until user stop or error
	err := storageProvider.RunWebServer()
	Abort(err)
}
