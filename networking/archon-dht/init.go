package archon_dht

import (
	"fmt"
	"strconv"

	//	"strconv"
	"time"

	archonCommon "github.com/itsmeknt/archoncloud-go/common"

	"github.com/libp2p/go-libp2p-core/peer"

	dht "github.com/itsmeknt/archoncloud-go/networking/archon-dht/mods/kad-dht-mod"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	golog "github.com/ipfs/go-log"

	gologging "github.com/whyrusleeping/go-logging"
)

func GetNodeID(seed int64) (peer.ID, error) {
	priv, err := GetRSAKey(seed)
	id, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", err
	}
	archonCommon.LogDebug.Println("NodeId ", id)
	return id, nil
}

func Init(configArr []DHTConnectionConfig, basePort int) (*ArchonDHTs, error) {
	archonCommon.LogDebug.Println("dht.Init()")
	if len(configArr) < 1 {
		return nil, fmt.Errorf("error Init: configArr must have non-zero length")
	}
	localPeerEndpoint = "http://localhost:" + strconv.Itoa(basePort) + "/api/v0/id"
	archonDHTs := new(ArchonDHTs)
	archonDHTs.Init()
	hasDefaultNonPermissionedLayer := false
	for i := 0; i < len(configArr); i++ {
		// SHOULD ADD VALIDATING CONFIG STEP// FIXME
		r, d, err := initEach(configArr[i])
		if err != nil && err.Error() != "" {
			archonCommon.LogDebug.Println("dht.Init() returns")
			return nil, err
		}
		archonDHTs.AddLayer(configArr[i].PermissionLayer.ID(), r, d, configArr[i])
		if configArr[i].PermissionLayer.ID() == "NON" {
			hasDefaultNonPermissionedLayer = true
		}
	}
	if !hasDefaultNonPermissionedLayer {
		return nil, fmt.Errorf("error Init: configArr must have config for NonPermissionedLayer")
	}
	pollAnnounceUrlInterval := 24 * time.Hour
	archonDHTs.pollAnnounceUrl(pollAnnounceUrlInterval)
	pollUpdateSPCacheInterval := 5 * time.Second
	archonDHTs.pollUpdateSPProfileCache(pollUpdateSPCacheInterval)
	// debug
	archonDHTs.RunTestTriggerServer()
	/*go func() {
		for {
			fmt.Println("peers")
			p := archonDHTs.Layers["ETH"].Peers()
			//p := archonDHTs.Layers["NON"].Peers()
	    for i := 0; i < len(p); i++ {
				fmt.Println("debug peer ", p[i])
			}
			time.Sleep(3 * time.Second)
		}
		} ()*/

	archonCommon.LogDebug.Println("dht.Init() returns")
	return archonDHTs, nil
}

func setAllLoggers() {
	loggingLevel := archonCommon.GetLoggingLevel()
	var glLevel gologging.Level
	switch loggingLevel {
	case archonCommon.LogLevelDebug:
		glLevel = gologging.DEBUG
		break
	case archonCommon.LogLevelTrace:
		glLevel = gologging.DEBUG
		break
	case archonCommon.LogLevelInfo:
		glLevel = gologging.INFO
		break
	case archonCommon.LogLevelWarning:
		glLevel = gologging.WARNING
		break
	case archonCommon.LogLevelError:
		glLevel = gologging.ERROR
		break
	default:
		glLevel = gologging.CRITICAL
	}
	golog.SetAllLoggers(glLevel)
}

func initEach(config DHTConnectionConfig) (*rhost.RoutedHost, *dht.IpfsDHT, error) {
	archonCommon.LogDebug.Println("dht.initEach() ", config.PermissionLayer.ID())
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	setAllLoggers()
	iamBootstrap := config.IAmBootstrap
	optInToNetworkLogging := config.OptInToNetworkLogging

	nodeID, err := GetNodeID(config.Seed)
	if err != nil {
		return nil, nil, err
	}
	// check if self is registered
	if config.PermissionLayer.Permissioned() {
		to := 5 * time.Second
		self := new(peer.AddrInfo)
		self.ID = nodeID
		peersArray := *new([]peer.AddrInfo)
		peersArray = append(peersArray, *self)
		validated, err := config.PermissionLayer.ValidatePeers(peersArray, to)
		// TODO ENSURE THAT NODEID IS REGISTERED W PERMLAYERADDRESS (MISMATCH IS POSSIBLE)
		if err != nil {
			return nil, nil, err
		}
		reg := true
		if len(validated) > 0 {
			if self.ID != validated[0].ID {
				reg = false
			}
		} else {
			reg = false
		}
		if !reg {
			return nil, nil, fmt.Errorf("error dht init, node and its associated nodeID must be registeredsp")
		}
	}

	// Make a host that listens on the given multiaddress
	var bootstrapPeers []peer.AddrInfo
	if config.Global {
		bootstrapPeers = convertPeers(config.BootstrapPeers)
	} else if !iamBootstrap {
		bootstrapPeers = getLocalPeerInfo()
	}
	ha, d, err := makeRoutedHost(config, bootstrapPeers)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		if optInToNetworkLogging {
			PollReportConnectionsToNetwork(*ha, config, 3600)
		}
	}()

	archonCommon.LogDebug.Println("dht.initEach() returns ", config.PermissionLayer.ID())
	return ha, d, nil
}
