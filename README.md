# Abstract

This software provides blockchain based storage. **Ethereum** and **Neo** blockchains are supported.  
It builds two executables: the Storage Provider (archonSP) and upload/download client (archon).

# Table of Contents
<!--ts-->
   * [Abstract](#abstract)
   * [Set-up](#set-up)
      * [Build](#build)
      * [Archon Cloud Centralized Endpoint](#archon-cloud-centralized-endpoint)
         * [Set Up](#set-up-1)
         * [Launch](#launch)
      * [Archon Cloud Service](#archon-cloud-service)
         * [Install](#install)
         * [Set Up](#set-up-2)
         * [Launch](#launch-1)
      * [Accounts](#accounts)
         * [Mode](#mode)
         * [Commands](#commands)
            * [Create](#create)
            * [Add](#add)
            * [Modify](#modify)
            * [View](#view)
            * [View All](#view-all)
            * [Backup](#backup)
   * [Usage](#usage)
      * [Archon Centralized Endpoint](#archon-centralized-endpoint)
         * [Upload](#upload)
         * [Download](#download)
      * [Archon Cloud Service](#archon-cloud-service-1)
         * [Upload](#upload-1)
         * [Download](#download-1)
   * [Architecture](#architecture)
      * [Implementation Notes](#implementation-notes)
         * [PUT vs POST for upload](#put-vs-post-for-upload)
         * [Admin interface security](#admin-interface-security)
<!--te-->

# Set-up

## Build

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

### Set Up

The software requires several keys and passwords which are secret. These instructions place them in the `etc/secret` directory to clearly label them and prevent accidental check-in to repo.

1. Create only-owner-accessible files with `(umask 0077 && touch etc/secret/archon.{key,pw})`.
1. Place encrypted secret key in `etc/secret/archon.key`.
1. Place key password in `etc/secret/archon.pw`.
1. Update `etc/secret/bunny-cdn.yml` with the real Bunny CDN access key.
1. Create admin interface certificate with `util/make-admin-cert.sh`.

Finally, create link to `$ARCHON_CLI` as `bin/archon`

### Launch

The daemon may be started with

    $ bin/aced etc/prod.yml

The software is ready when it outputs `listening on https://localhost:8800`. It may be stopped with `Ctrl-C` or `kill`.

## Archon Cloud Service

`acsd` (Archon Cloud Service daemon) is the front-end software that provides an HTTP interface to several cloud storage services (currently AWS S3, Backblaze B2, and Digital Ocean Spaces).

### Install

The software is distributed as a docker image. It can be installed with the command

    $ docker pull devopsarchoncloud/acs:latest

Additionally, although not required, the `docker-compose.yml` in the project's repo file can be used to make operating the service easier. This document assumes it is installed.

### Set Up

The service uses the directory `$HOME/data` to hold various configuration files.

    data
    +-- etc
    |   +-- accounts.db
    |   +-- credentials.yml
    |   +-- secret
    |   |   +-- acs.key -> /etc/letsencrypt/live/acs.archon.cloud/privkey.pem
    |   +-- tls
    |       +-- acs.crt -> /etc/letsencrypt/live/acs.archon.cloud/fullchain.pem
    +-- var
        +-- objects.db
        +-- tmp

All but the `objects.db` and `accounts.db` file need to be created before the service is started.

### Launch

The daemon can be started with the command

    $ docker-compose up -d

and stopped with the command

    $ docker-compose down

## Accounts

Uploads are restricted to users having a valid account. The account is identified by supplying a `?token=<id>` parameter with the upload request.

The location of the account database is specified in the config file and is `etc/accounts.db` by default. The `bin/acectl` program is used to work with the account file. Only users having appropriate permissions may run it.

Most `acectl` commands have the following format:

    $ bin/acectl [global options] <command> [command options] [command arguments]

Usage information can also be viewed using the `--help` option.

### Mode

The  program supports two modes, offline and online. Online mode is used unless `--file=<path/to/db>` is given as a global option.

Offline mode must be used to access an account file that is not in use. It is useful for initial setup and for potentially long-running operations like `viewall`.

Online mode must be used to access the account file that is in use by `aced` or `acsd`. It is useful for adding or modifying accounts while the service is running without requiring service interruption.

### Commands

#### Create

The create command is used to create a new, empty accounts database file. It only works in offline mode, so the `--file=<path/to/db>` global option is required.

#### Add

The `add` command is used to add a new account to the database. It accepts no arguments and supports the following options:

* `--id=<str>`: Specify an account id (token) instead of creating one randomly. Must consist of between 6 and 64 letters, numbers, or dashes.
* `--owner=<str>`: String used to identify owner. Defaults to `none`.
* `--size=<spec>`: Maximum size per upload. There is no limit on total upload amount. Defaults to `128MiB`.
* `--bucket=<name>`: Top-level-directory under which files must be uploaded. Required.
* `--comment=<str>`: Any comments. Defaults to empty string.
* `--enabled=<true|false>]`: Allow uploads for account. Defaults to `true`.

#### Modify

The `modify` command is used to change an existing account in the database. It accepts a single argument, the id of the account to be changed, and supports the same options as the `add` command with the exception of `--id`, which cannot be changed.

Changes apply only to future uploads. If an account is disabled, any files already uploaded will remain available for download. Likewise, if the maximum upload size or root path is changed, existing files will not be affected.

#### View

The `view` command is used to view an account in the database. It accepts a single argument, the id of the account to be viewed, and accepts no options.

#### View All

The `viewall` command is used to view all accounts in the database. It accepts no arguments and supports the following options:

* `--no-header`: Do not include column labels in output.

#### Backup

The `backup` command is used to create a copy of an account database. This is preferred over simply copying the file for the following reasons:

1. If the file is in use and an update is in progress, the resulting copy may be in an inconsistent state. The backup command writes a consistent snapshot of the database.
1. An in-use database may contain old versions of modified account data. The backup command writes only the most recent versions of accounts and produces as smaller file.

# Usage

## Archon Centralized Endpoint

### Upload

Uploads are restricted to clients having an enabled account. Files may be uploaded to the Archon Cloud (and IPFS cloud) using a simple HTTP PUT request. For example, if the account `L6US54KBU3DGNB5CPXEXCNUQKTOUASPA` owns the `demo` bucket, the file `~/doc/dracula.txt` may be uploaded to `/demo/dracula.txt` from the shell with

    $ curl --silent -T ~/doc/dracula.txt 'https://upload.archon.cloud/demo/dracula.txt?token=L6US54KBU3DGNB5CPXEXCNUQKTOUASPA' | jq
    {
      "cdn_url": "http://edge.archon.cloud/stories/dracula.txt",
      "archon_url": "arc://ArchonDemo.af/stories/dracula.txt",
      "archon_hash": "arc://ArchonDemo.af/:2pvHch0Fnr64NrwuK2wa1t5+B8hx+cX5YbrVzGPWNU91FWvqVmz8slN/iKebrauXHAKTuL66KhJ+v2nOclygFA==:.txt",
      "ipfs_hash": "QmWQfVRwbyweEXXhZNbEs9JCMEu3tJ613GqyHb8CkB91JK"
    }

The response is a JSON object containing the CDN URL, Archon URL, Archon hash, and IPFS hash.

The file size limit for uploads is account-specific, but defaults to 128MiB and may be set to a maximum of 256MiB.

The supported URL params on upload are:

* `token=<id>`: The account id.
* `permanent=true`: Pin the IPFS upload.

### Download

Files may be downloaded from the Archon Cloud using a simple HTTP GET request. For example, the cloud file `/demo/dracula.txt` may be downloaded from the shell with

    $ wget https://edge.archon.cloud/demo/dracula.txt

Unlike uploads, downloads may be done by anyone, not just the owner of the bucket.

## Archon Cloud Service

### Upload

Uploads are restricted to clients having an enabled account. Files may be uploaded to all the backend-end clouds simultaneously using a simple HTTP PUT request or an AWS S3 client.

For example, the file `~/doc/dracula.txt` may be uploaded to `/stories/dracula.txt` from the shell with

    $ curl --silent -T ~/doc/dracula.txt 'https://acs.archon.cloud/stories/dracula.txt?token=L6US54KBU3DGNB5CPXEXCNUQKTOUASPA'

AWS S3 clients require a credentials file to be set up, usually in `~/.aws/credentials`. The `aws_access_key_id` should be set to the account token, and the `aws_secret_access_key` should be set to an empty string.

Additionally, AWS S3 clients require a storage "bucket" to be specified. This bucket is always `archon`. For example, the file `~/doc/dracula.txt` may be uploaded to `/stories/dracula.txt` from the shell with

    $ aws s3 cp --endpoint=http://acs.archon.cloud ~/doc/dracula.txt 's3://archon/stories/dracula.txt'

Responses for successful requests have no body. On error, and S3-like XML error response will be returned.

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
