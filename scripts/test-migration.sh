#!/bin/bash

set -eu

source ./scripts/setup-chain.sh

current_dir=$(pwd)
TMP_ROOT="$(dirname $(pwd))/_build"
mkdir -p "${TMP_ROOT}"

TMP_BUILD="${TMP_ROOT}"/jprovd

install_old () {
    TMP_GOCACHE="${TMP_ROOT}"/gocache
    mkdir -p "${TMP_GOCACHE}"

    curl -o v1.1.2.zip --output-dir "${TMP_ROOT}" \
        -L https://github.com/JackalLabs/canine-provider/archive/refs/tags/v1.1.2.zip 

    unzip -u -d "${TMP_ROOT}" "${TMP_ROOT}"/v1.1.2.zip

    cd "${TMP_ROOT}"/canine-provider-1.1.2

    go install -mod=readonly ./jprov/jprovd
    
    cd "${current_dir}"
    jprovd version
}

install_new () {
    make install
    jprovd version
}

start_chain () {
    startup
    from_scratch
    fix_config

    screen -d -m -S "canined" bash -c "canined start --pruning=nothing --minimum-gas-prices=0ujkl"
}

restart_chain () {
    screen -d -m -S "canined" bash -c "canined start --pruning=nothing --minimum-gas-prices=0ujkl"
}

start_provider () {
    screen -d -m -L -Logfile "provider$3.log" \
        -S "provider$3" bash -c "./scripts/start-provider.sh $1 $2 $3"
}

restart_provider () {
    screen -d -m -L -Logfile "provider$1.log" \
        -S "provider$1" bash -c "./scripts/restart-provider.sh $1"
}

migrate_provider () {
    echo y | jprovd migrate --home="$HOME/providers/provider$1"
}

sender="jkl10k05lmc88q5ft3lm00q30qkd9x6654h3lejnct"

upload_file () {
    resp=$(curl -v -F sender=$sender -F file=@$1 http://localhost:333$2/upload)

    # get cid value from json respnose and strip double quote at front and end 
    # example:
    # {"cid":"jklc1amfnkh8fj8wpvadxp8zjm4h3kgnr0m6qqk5a7dkt0a87pc3yc6nqq4sawe","fid":"jklf12g2ae3tw5397rjehjavcfxzxp4nu9nggpm6lvs6m9wfns0gs3ecqpxt6vq"}
    # gets:
    # jklc1amfnkh8fj8wpvadxp8zjm4h3kgnr0m6qqk5a7dkt0a87pc3yc6nqq4sawe
    cid=$(echo "$resp" | jq '.cid' | sed 's/"//g')
    fid=$(echo "$resp" | jq '.fid' | sed 's/"//g')

    sleep 5

    canined tx storage sign-contract "$cid" --from charlie -y

    echo "$1 uploaded... fid: ${fid}"
    sleep 5
}

shut_down () {
    killall canined jprovd
}

install_old

start_chain
echo "CHAIN STARTED!!!"
sleep 5

start_provider 54f86a701648e8324e920f9592c21cc591b244ae46eac935d45fe962bba1102c \
    jkl1xclg3utp4yuvaxa54r39xzrudc988s82ykve3f 0
#start_provider a29c5f0033606d1ac47db6a3327bc13a6b0c426dbfe5c15b2fcd7334b4165033 \
#    jkl1tcveayn80pe3d5wallj9kev3rfefctsmrqf6ks 1
#start_provider a490cb438024cddca16470771fb9a21938c4cf61176a46005c6a7b25ee25a649 \
#    jkl1eg3gm3e3k4dypvvme26ejmajnyvtgwwlaaeu2y 2
#start_provider 6c8a948c347079706e404ab48afc5f03203556e34ea921f3b132f2b2e9bcc87d \
#    jkl1ga0348r8zhn8k4xy3fagwvkwzvyh5lynxr5kak 3
#start_provider 8144389a23c6535e276068ff9043b2b6ff95aa3c103c35486c8f2d2363606fd5 \
#    jkl18encuf0esmxv3pxqjqvn0u4tgd6yzuc8urzlp0 4
#start_provider 0e019088a0fafa8f77cb5c0d0f6cb6b63a0015f20d2450480cbcdee44d170aab \
#    jkl1sqt9v0zwwx362szrek7pr3lpq29aygw06hgyza 5
#start_provider adf5a86ac54146b172c20b865c548e900c51439c3723af14aeab668ccd2b8ecf \
#    jkl1yu099xns2qpslvyrymxq3hwrqhevs7qxksvu8p 6
echo "provider started!!!"
sleep 30

# upload files
canined tx storage buy-storage jkl10k05lmc88q5ft3lm00q30qkd9x6654h3lejnct 720h 3000000000 ujkl --from charlie -y
sleep 5

upload_file ./scripts/dummy_data/1.png 0
#upload_file ./scripts/dummy_data/2.png 1
#upload_file ./scripts/dummy_data/3.png 6
upload_file ./scripts/dummy_data/4.png 0
upload_file ./scripts/dummy_data/5.svg 0
upload_file ./scripts/dummy_data/6.wav 0
upload_file ./scripts/dummy_data/test.txt 0

sleep 10


echo "shutting down providers..."
killall jprovd

sleep 4

echo "upgrading provider..."
install_new

read -rsp $'Press any key to shutdown and upgrade provider...\n' -n1 key
#migrate_provider 0
#migrate_provider 1
#migrate_provider 2
#migrate_provider 3
#migrate_provider 4
#migrate_provider 5
#migrate_provider 6
#
#
#sleep 5
#
#restart_provider 0
#restart_provider 1
#restart_provider 2
#restart_provider 3
#restart_provider 4
#restart_provider 5
#restart_provider 6


read -rsp $'Press any key to shutdown...\n' -n1 key

shut_down
#cleanup
