package archon_dht

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/itsmeknt/archoncloud-go/common"
	"github.com/libp2p/go-libp2p-core/peer"

	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/pariz/gountries"
)

type ReportConnectionsLog struct {
	NodeId        peer.ID         `json: "nodeid`         // type will be specific to the id of the host
	ConnectionIds []peer.ID       `json: "connectionids"` // type will be specific to the ids of connections
	Time          time.Time       `json: "time"`
	Address       []byte          `json: "address"`
	CountryCode   gountries.Codes `json: countrycode`
}

type WrappedLog struct {
	Log ReportConnectionsLog
}

func PollReportConnectionsToNetwork(host rhost.RoutedHost, config DHTConnectionConfig, interval time.Duration) {
	clUrl := "dhtlogger.archon.cloud"
	clUrl = "https://" + clUrl + "/centralLogging"
	report := new(ReportConnectionsLog)
	report.NodeId = host.ID()
	var address []byte
	if config.Account != nil {
		address = append(address, config.Account.AddressBytes()...)
	}
	report.Address = append(report.Address, address...)
	report.CountryCode = config.CountryCode
	for {
		// construct ReportConnectionsLog logData
		report.Time = time.Now()
		peers := host.Peerstore().Peers()
		report.ConnectionIds = *new([]peer.ID)
		for i := 0; i < len(peers); i++ {
			report.ConnectionIds = append(report.ConnectionIds, peers[i])
		}
		wrappedReport := WrappedLog{Log: *report}
		go func(data WrappedLog) {
			// network call POST
			postString, _ := json.Marshal(data)
			var reqBytes = []byte(postString)
			req, err_req := http.NewRequest("POST", clUrl, bytes.NewBuffer(reqBytes))
			if err_req != nil {
				common.LogError.Println(err_req)
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: time.Second * 10}
			resp, err_resp := client.Do(req)
			if err_resp != nil {
				common.LogError.Println(err_resp)
			}
			if resp != nil {
				if resp.Body != nil {
					resp.Body.Close()
				}
			}
		}(wrappedReport)
		time.Sleep(interval * time.Second)
	}
}
