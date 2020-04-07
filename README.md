# Abstract

This software provides blockchain based storage. **Ethereum** and **Neo** blockchains are supported.  
It builds two executables: the Storage Provider (archonSP) and upload/download client (archon).

# Table of Contents
<!--ts-->
   * [Abstract](#abstract)
   * [Build](#build)
   * [Usage](#usage)
      * [Storage Provider](#storage-provider)
      * [Archon Cloud Service](#archon-cloud-service-1)
         * [Upload](#upload-1)
         * [Download](#download-1)
   * [Architecture](#architecture)
      * [Implementation Notes](#implementation-notes)
         * [PUT vs POST for upload](#put-vs-post-for-upload)
         * [Admin interface security](#admin-interface-security)
<!--te-->

# Build

    $ git clone https://github.com/archoncloud/archoncloud-go
    $ cd archoncloud-go
    $ make

Go compiler V1.14 or higher is needed.  
Note that in order to use "make" on Windows you need to have MinGW installed.  
The GOBIN environment variable will be used, if set.  
Alternatively, you can build using the "go" command.  

    $ cd archoncloud-go/cmd/archonSP
    $ go build . -i -o archonSP
    $ cd archoncloud-go/cmd/archon
    $ go build . -i -o archon

On Windows specify the output as archonSP.exe and archon.exe.  

# Usage
## Storage Provider

The storage provider executable (archonSP or archonSP.exe) is a server that needs to run continually.  
Best is placed in a separate folder. When first started will create several sub-folder and a default configuration file `archonSP.config`.  
This file has some items that will need to be changed to fit your setup.  
Many of these can be set from the command line. Start with `--help` to see what they are. Once entered, they will be remembered in the config file and don't need to be entered again.  
The config file is a JSON file that can be edited with a text editor.  
You can participate in either the Ethereum or Neo blockchain, or both, by entering the wallet path for the approriate wallet file.  
You will be asked for a password once the program starts.  
You can also create new wallets from the command line, but you will need to add funds to it before using.
Some items in this file can only be changed with an editor, not from the command line.  
These are:  
- `eth_rpc_urls`: Only needed if you entered an Ethereum wallet. One or more URLs that provide an Ethereum RPC service. Enter the one you want of use. Infura is one such provider of RPC connectivity.  
- `neo_rpc_urls`:  Only needed if you entered a Neo wallet. One or more URLs that provide a NEO RPC service. The config file will be populated with defaults, but you can change as needed.  
- `bootstrap_peers`, `eth_bootstrap_peers`, `neo_bootstrap_peers`: These are the DHT bootstraps. Best to leave as they are, but can be changed if you know what you are doing.  
 

### Download

Files may be downloaded from the cloud with a simple HTTP GET request. For example, the cloud file `/stories/dracula.txt` may be downloaded from the shell with

    $ wget https://acs.archon.cloud/stories/dracula.txt

Using an AWS S3 client, the same file may be downloaded with

    $ aws s3 cp --endpoint=https://acs.archon.cloud 's3://archon/stories/dracula.txt' ./dracula.txt

For naive clients like curl and wget, the download response is an HTTP redirect to the document in one of the back-end clouds. For aws clients, the response is the document itself since aws clients do not handle the redirect.

Files may be downloaded using either method regardless of the method used to upload them.

# Architecture

Below is a rough diagram of the Archon Centralized Endpoint system architecture.

           +--------+                                   +-----------+
           |  User  |                                   | Publisher |
           +--------+                                   +-----------+
               |^                                            |^
               ||                                            || CDN URL
           GET || Document                               PUT || Archon URL
               ||                                            || IPFS URL
               v|                                            v|
    +-----------------------+        GET        /=============================\     +-------+
    |          CDN          | ----------------> | Archon Centralized Endpoint | --> | Local |
    |   edge.archon.cloud   | <---------------- |   ace|upload.archon.cloud   | <-- | Cache |
    +-----------------------+     Document      \=============================/     +-------+
                                                               ^
                                                               |
                                                        Upload | Download
                                                               |
                                                              / \
                                                     +--------   --------+
                                                     |                   |
                                                     v                   v
                                                 +~~~~~~~~+          +~~~~~~~+
                                                 | Archon |          | IPFS  |
                                                 | Cloud  |          | Cloud |
                                                 +~~~~~~~~+          +~~~~~~~+

## Implementation Notes

### PUT vs POST for upload

The system uses PUT rather than POST for uploading files because that's the correct method to use according to the HTTP specification (RFC 7231). The POST method is used to submit data (e.g. form contents) to the resource at the given URI. The PUT method, on the other hand, is used to create or replace the resource itself.

### Admin interface security

Only users having appropriate privileges may run admin commands. In offline mode, this means users who have read and/or write permissions on the account database file. In online mode, it is users on the localhost who have read permission on the key and certificate file used to encrypt communication between the `acectl` client and `aced` server. In either case, we rely on the Linux kernel to enforce these permissions.

A new admin interface key and certificate is issued with each releases, so a client will not be able to communicate with a server from a different release.
