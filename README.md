![Jackal Provider Cover](./assets/jklstorage.png)
# Jackal Storage Provider

[![Build](https://github.com/JackalLabs/canine-provider/actions/workflows/build.yml/badge.svg)](https://github.com/JackalLabs/canine-provider/actions/workflows/build.yml)
[![Test](https://github.com/JackalLabs/canine-provider/actions/workflows/test.yml/badge.svg)](https://github.com/JackalLabs/canine-provider/actions/workflows/test.yml)
[![golangci-lint](https://github.com/JackalLabs/canine-provider/actions/workflows/golangci.yml/badge.svg)](https://github.com/JackalLabs/canine-provider/actions/workflows/golangci.yml)

## Overview
The storage provider is a web server that accepts incoming files from users and creates contracts for the users to approve. These contracts last until the user either cancels them or the provider itself goes offline.

## API

You can explore the provider API [here](https://www.postman.com/navigation-pilot-71533452/workspace/jackal-storage-api)

## Quickstart
This assumes you have either already set up a node or are using another RPC provider in your `~/.jackal-storage/config/client.toml` file.

To quickly set up a storage provider, one must initialize their provider & announce themselves to the network. Then they start the provider from their account of choice which stores files in the `~/.jackal-storage/storage` folder (this can be changed with the --home flag).

```sh
$ jprovd client config chain-id {current-chain-id}

$ jprovd client gen-key

$ jprovd init {IP_ADDRESS} {STORAGE_IN_BYTES} {KEYBASE_IDENTITY}

$ jprovd start
```

## Posting files
Files can be uploaded through a POST request to `localhost:3333/upload` with form data.
### Form Data
| Key    | Data       |
|--------|------------|
| file   | {filedata} |
| sender | {address}  |

### Response
The response will be a JSON response formatted as:
```JSON
{
    "CID": "jklc1...",
    "FID": "jklf1..."
}
```

## Getting files
Gettings files is as easy as running a GET request at `localhost:3333/download/{FID}`. This will return the file as a blob to the browser.

