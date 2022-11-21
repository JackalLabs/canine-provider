![Jackal Provider Cover](./assets/jklstorage.png)
# Jackal Storage Provider

[![Build](https://github.com/JackalLabs/canine-provider/actions/workflows/build.yml/badge.svg)](https://github.com/JackalLabs/canine-provider/actions/workflows/build.yml)
[![Test](https://github.com/JackalLabs/canine-provider/actions/workflows/test.yml/badge.svg)](https://github.com/JackalLabs/canine-provider/actions/workflows/test.yml)

## Overview
The storage provider is a web-server that accepts incoming files from users and creates contracts for the users to approve. These contracts last until the user either cancels them or the provider itself goes offline.

## API

You can explore the provider API [here](https://www.postman.com/navigation-pilot-71533452/workspace/jackal-storage-api)

## Quickstart
This assumes you have either already set up a node or are using another RPC provider in your `~/.jprovd/config/client.toml` file.

To quickly set up a storage provider, one must initialize their provider & announce themselves to the network. Then they start the provider from their account of choice which stores files in the `~/.jprovd/config/networkfiles` folder (this can be changed with the --home flag).

To use a different home directory to store the files, you must run `init` with the home flag set to where you wish to initialize it.

You must also be using the keyring-backend: `test`. (`jprovd config keyring-backend test`)

```sh
$ jprovd config chain-id {current-chain-id}

$ jprovd tx storage init-miner {IP_ADDRESS} {STORAGE_IN_BYTES} --from {KEY_NAME} --gas-prices=0.002ujkl --gas-adjustment=1.5

$ jprovd start-miner --from {KEY_NAME} --gas-prices=0.002ujkl --gas-adjustment=1.5 -y
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

