# archon-go-dht

#### A client's participation in the dht is initialized using 

```
Init(configArr []DHTConnectionConfig, basePort int) (*ArchonDHTs, error)
```


#### An application can participate simultaneously in disjoint dhts. A practical implementation would be participation in a permissioned and unpermissioned dht. Simply call `Init` with different configs and permission layers. A client can participate in any number of disjoint permissioned dhts.


#### For each permission layer, populate a config. Collate these configs in a config array, then run `Init(..)` with this config array as input
 
```
  var basePort int = <some integer>

  eth := new(dht.Ethereum);
  permissionedConfigDHT := dht.DHTConnectionConfig{
    Seed: *seed,
    Global: *global,
    IAmBootstrap: *iamBootstrap,
    MyWalletPath: *myWalletPath,
    MyWalletPassword: *myWalletPassword,
    OptInToNetworkLogging: *optInToNetworkLogging,
    SelfReportedCountryCode: *selfReportedCountryCode,
    PermissionLayer: *eth,
    Url: string("!exampleUpload123:123#532"), 
    MyPartialMultiAddress: "/ip4/<nodes_public_ip>/tcp/" + strconv.Itoa(basePort + 1),
    BootstrapPeers: []string{
        "/ip4/148.64.103.83/tcp/4002/ipfs/QmNX6ASyukLch38D2Z1h4cMh39ATfqqDom1xJWv2YHc1eG"}}
  
  nonPermissioned := new(dht.NonPermissioned);
  freeConfigDHT := dht.DHTConnectionConfig{
    Seed: (*seed + 1),
    Global: *global,
    IAmBootstrap: *iamBootstrap,
    MyWalletPath: *myWalletPath,
    MyWalletPassword: *myWalletPassword,
    OptInToNetworkLogging: *optInToNetworkLogging,
    SelfReportedCountryCode: *selfReportedCountryCode,
    PermissionLayer: *nonPermissioned,
    Url: string("!exampleUpload123:333#222"),
    MyPartialMultiAddress: "/ip4/<nodes_public_ip>/tcp/" + strconv.Itoa(basePort + 2),
    BootstrapPeers: []string{
        "/ip4/148.64.103.83/tcp/4002/ipfs/QmNX6ASyukLch38D2Z1h4cMh39ATfqqDom1xJWv2YHc1eG"}}

  var configArray []dht.DHTConnectionConfig
  configArray = append(configArray, permissionedConfigDHT)
  configArray = append(configArray, freeConfigDHT)

  aDht, err := dht.Init(configArray, basePort);
```

#### The config struct is defined as

```
type DHTConnectionConfig struct {
	Seed                    int64  // seed to initialize fresh dht rsa keyset and id
	Global                  bool   // bootstrap to global set ?
	IAmBootstrap            bool   // declare if self is a bootstrap node
	MyWalletPath            string // filename of wallet for permission layer
	MyWalletPassword        string // password of wallet for permission layer
	OptInToNetworkLogging   bool   // self-explanatory
	SelfReportedCountryCode string // optional self-reported country code using //ISO_ALPHA2_COUNTRY_CODES
	PermissionLayer         PermissionLayer
  Url                     string // this should be copied from archonSP config 

	MyPartialMultiAddress string
	BootstrapPeers        []string
}
```

#### Q: "Once initialized, what can I do?"

#### A: Now that we have instances of routed host, we have access to the api provided by go-libp2p-kad-dht, but the archon wrapper supplies some purpose-built api endpoints listed in the following examples.
 
##### Example of getting your node's ID

```
  nodeID := GetNodeID(*dhtConfig) // this is useful to fill registerSP struct for "NodeID"
```


##### Example of broadcasting to network that self is providing specific objects

```
  var stored string = "exampleStoredStringRepresentation" 
  // in practice "exampleStoredStringRepresentation" will look like
  // "<namedArchonUrl> + <shardIdx>" or "<hashedArchonUrl> + <shardIdx>" 
  err := aDht.Stored(stored)
```

##### Example of getting urls of nodes holding specific objects

```
    var stringArr []string = []string{"example1", "example2", "example3"}
    permissionLayer2UrlArray, err := aDht.GetUrlsOfNodesHoldingKeysFromAllLayers(stringArr, 3)  
    // 3 is timeout in seconds

    // note the type: type PermissionLayer2UrlArray map[string][]string

```

###### The above query can be made specific to permission layer

```
    var stringArr []string = []string{"example1", "example2", "example3"}
    urlArray, err := aDht.Layers["ETH"].GetUrlsOfNodesHoldingKeys(stringArr, timeout) 
    // timeout is time.Duration timeout in seconds
```

###### Get a collection of RegisteredSP profiles that point to storage providers to the marketplace but narrowed to a specific permissionLayer

```

    permissionLayer := dht.PermissionLayerID("ETH")
    res, err := aDht.GetArchonSPProfilesForMarketplace(permissionLayer)

```

###### Given an array of NodeIDs, retrieve from the dht their corresponding urls

```
    preferredPermissionLayer := dht.PermissionLayerID("ETH")
    timeoutInSeconds := time.Duration(3)
    urls, err := aDht.GetUrls(nodeIDs, permissionLayer, timeoutInSeconds)

```

##### Simple example of finding a peer and connecting

```
      // example of finding peer and connecting
      // this is not really necessary for the archonSP to use, but
      // may serve useful for other purposes
      peerId, _ := peer.IDB58Decode(*target);
      peerAddrInfo, _ := d.FindPeer(context.Background(), peerId);
      // connect here
      p := peerAddrInfo
      if err := aDht.Layers["ETH"].Connect(context.Background(), p); err != nil {
        fmt.Println("bootstrapDialFailed", p.ID)
        fmt.Printf("failed to bootstrap with %v: %s", p.ID, err)
      } else {
        // THIS SHOWS MUTUAL CONNECTION
        fmt.Println("bootstrap dial success ", p.ID);
        peers := aDht.Layers["ETH"].Peers();
        for i := 0; i < len(peers); i++ {
          fmt.Println("debug ", peers[i]);
        }
      }
```


##### These are methods made available from the underlying libp2p kademlia api

##### To be consistent with the above, these methods would be called like `aDht.Layers["ETH"].dHT.Method(...)`

```
// Kademlia 'node lookup' operation. Returns a channel of the K closest peers
// to the given key
func (dht *IpfsDHT) GetClosestPeers(ctx context.Context, key string) (<-chan peer.ID, error)

// PutValue adds value corresponding to given Key.
// This is the top level "Store" operation of the DHT
func (dht *IpfsDHT) PutValue(ctx context.Context, key string, value []byte, opts ...routing.Option) (err error)

// GetValue searches for the value corresponding to given Key.
func (dht *IpfsDHT) GetValue(ctx context.Context, key string, opts ...routing.Option) (_ []byte, err error)


// GetValues gets nvals values corresponding to the given key.
func (dht *IpfsDHT) GetValues(ctx context.Context, key string, nvals int) (_ []RecvdVal, err error)

// Provide makes this node announce that it can provide a value for the given key
func (dht *IpfsDHT) Provide(ctx context.Context, key cid.Cid, brdcst bool) (err error)

// FindProviders searches until the context expires.
func (dht *IpfsDHT) FindProviders(ctx context.Context, c cid.Cid) ([]peer.AddrInfo, error)
// ASYNC VERSIONS AVAILABLE AS WELL

// FindPeer searches for a peer with given ID.
func (dht *IpfsDHT) FindPeer(ctx context.Context, id peer.ID) (_ peer.AddrInfo, err error)

// FindPeersConnectedToPeer searches for peers directly connected to a given peer.
func (dht *IpfsDHT) FindPeersConnectedToPeer(ctx context.Context, id peer.ID) (<-chan *peer.AddrInfo, error)
```


#### Example configurations for

  -Bootstrap node
```
    IamBootstrap true
    Seed 123 
    MyEthWalletPassword ethTestingWallet2 
    MyEthWalletPath ethTestingWallet2.json 
    IamCentralLogger true
```

  -(Demo) 2nd node 
```
    Global true
    Seed 1232 
    MyEthWalletPath ethTestingWallet0.json 
    MyEthWalletPassword ethTestingWallet0 
    OptInToNetworkLogging true
```

  -(Demo) 3rd node
```
    Global true 
    Seed 1233 
    MyEthWalletPath ethTestingWallet1.json 
    MyEthWalletPassword ethTestingWallet1 
    Target QmfYqARY2X47Vc4nAc82tm9ZcFf22aRv3WxW2cMiTRysRg // this is example dht id of stream recipient
    OptInToNetworkLogging true
```

## Appendix

