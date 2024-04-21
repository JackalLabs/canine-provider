FROM golang:1.22 as build

RUN apt update -y
RUN apt upgrade -y
RUN apt install build-essential -y
#RUN apk add --no-cache git make gcc musl-dev linux-headers ca-certificates build-base

ADD . canine-provider
WORKDIR canine-provider

RUN LEDGER_ENABLED=false make install

ENV PROVIDER_CHAIN_ID="jackal-1"
ENV PROVIDER_RPC="https://rpc.jackalprotocol.com:443"
ENV PROVIDER_DOMAIN="http://127.0.0.1:3333"
ENV PROVIDER_SPACE="1000000000000"
ENV PROVIDER_NAME="storage-provider"

RUN jprovd client gen-key
RUN jprovd client config chain-id $PROVIDER_CHAIN_ID
RUN jprovd client config node $PROVIDER_RPC

CMD ["sh", "scripts/docker.sh"]


