# Abstract

This software provides blockchain based storage, the *Archon Blockchain Storage*. **Ethereum** and **Neo** blockchains are supported.  
It builds two executables: the Storage Provider (**archonSP**) and upload/download client (**archon**). Tested to work on Linux and Windows.  
Storage providers can sign up and offer storage capacity. Users of the Archon client can then upload and download files from this distributed storage.  
Storage providers are paid in the currency of the blockchain. Users pay in the currency of the blockchain (for uploads, downloads are free).  
Sharded files are first packed in a container that includes a signed hash, so that integrity can be verified.

# Table of Contents
<!--ts-->
   * [Abstract](#abstract)
   * [Build](#build)
   * [Usage](#usage)
      * [Storage Provider](#storage-provider)
      * [Client](#client)
         * [Upload](#upload)
         * [Download](#download)
<!--te-->

# Build

    $ git clone https://github.com/archoncloud/archoncloud-go
    $ cd archoncloud-go
    $ make

Go compiler V1.14 or higher is needed.  
Note that in order to use "make" on Windows you need to have MinGW installed.  
The **GOBIN** environment variable will be used, if set.  
Alternatively, you can build using the "go" command.  

    $ cd archoncloud-go/cmd/archonSP
    $ go build . -i -o archonSP
    
    $ cd archoncloud-go/cmd/archon
    $ go build . -i -o archon

On Windows specify the output (-o) as archonSP.exe and archon.exe.  

# Usage
## Storage Provider

The storage provider executable (*archonSP* or *archonSP.exe*) is a server that needs to run continually.  
Best to place in a separate folder. When first started will create several sub-folder and a default configuration file `archonSP.config`.  
This file has some items that will need to be changed to fit your setup.  
Many of these can be set from the command line. Start with `--help` to see what they are. Once entered, they will be remembered in the config file and don't need to be entered again.  
The config file is a JSON file that can be edited with a text editor.  
You can participate in either the Ethereum or Neo blockchain, or both, by entering the wallet path for the appropriate wallet file.  
You will be asked for passwords once the program starts.  
You can also create new wallets from the command line, but you will need to add funds to them before using.  
The `host` entry needs to be set to a public IP or DNS name.
Some items in the config file can only be changed with an editor, not from the command line.  
These are:  
- `eth_rpc_urls`: Only needed if you entered an Ethereum wallet. One or more URLs that provide an Ethereum RPC service. Enter the one you want of use. Infura is one such provider of RPC connectivity.  
- `neo_rpc_urls`:  Only needed if you entered a Neo wallet. One or more URLs that provide a NEO RPC service. The config file will be populated with defaults, but you can change as needed.  
- `bootstrap_peers`, `eth_bootstrap_peers`, `neo_bootstrap_peers`: These are the DHT bootstraps. Best to leave as they are, but can be changed if you know what you are doing.  

On first run, a `registration.txt` file is also created with default entries.  
This will be used to register with the blockchain. Fill in the empty items with a text editor.  
The most important are the min ask values `GasPerGByte` and `WeiPerByte`. These will be used by the network when picking storage providers to upload to.  
Note that for Neo, the pay is in *CGAS*, not *GAS*, but *CGAS* is convertible one to one to *GAS*.
Once registered, you can re-register at a later time with different values, if needed.

## Client

The client executable (*archon* or *archon.exe*) can be used for an upload or download of a file to the Archon Blockchain Storage.  
Start with `--help` to see commands and options.
Once entered, options will be remembered in the config file, `archon.config`.  
The config file is a JSON text file that can be edited with a text editor.  
You will need an Ethereum or Neo wallet file, depending on which blockchain storage providers you wish to use (upload to or download from).
There are two items in the file that can only be edited with an editor (no command line options):  
- `eth_rpc_urls`: Only needed if you entered an Ethereum wallet. One or more URLs that provide an Ethereum RPC service. Enter the one you want of use. Infura is one such provider of RPC connectivity.  
- `neo_rpc_urls`:  Only needed if you entered a Neo wallet. One or more URLs that provide a NEO RPC service. The config file will be populated with defaults, but you can change as needed.  

There are several ways in which a file can be uploaded. This is controlled by the `encoding` option:
- `none`: as a whole file, to one storage provider only.
- `mxor`: Archon proprietary sharding. Creates 6 shards, of which at least 2 are needed for reconstructing the whole file.  
- `RSa`: Reed-Solomon archive optimized sharding. You specify the total number of shards and the required number.  
- `RSb`: Reed-Solomon browser optimized sharding. You specify the total number of shards and the required number.

`RSa` contains more metadata for allowing reconstruction of partially damaged shards. Aimed for long term archiving.  
Sharding allows the original file to be reconstructed even if some storage providers that stored the shards are offline or have disappeared.  
Sharded data will be uploaded to different storage providers. The aim is that no storage providers hold more than one shard of a file. But, if not enough storage providers are present, some may get more than one shard.  
Neo payments for upload are in *CGAS* not *GAS*. These are convertible one to one, but *CGAS* is the only payment form that works in Neo smart contracts. Downloads are free.  

### Upload
The first time you try the client it will ask for a user name and register this user name with the blockchain.
Example upload:  

    $ ./archon upload -f=mycat.jpg -e=mxor -t=named
    
After the upload completes, the client will print the Archon url for the stored file.   
In this case it will be `arc://upl3.eth.n2:6/mycat.jpg`.  
This was with the user name being "upl3". You need to store this string since this will be the only way to download the file.

### Download
Example download for the above Archon Url:  

    $ ./archon download -u="arc://upl3.eth.n2:6/mycat.jpg"
