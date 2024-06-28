#!/bin/bash

set -eu

source ./scripts/setup-chain.sh

HALT_UPGRADE_HEIGHT="30"
UPGRADE_HEIGHT="90"

OLD_CHAIN_VER="v3.2.2"
HALT_CHAIN_VER="marston/v3-halt"
NEW_CHAIN_VER="marston/v4-smodule-update"

OLD_PROVIDER_VER="v1.1.2"

sender="jkl10k05lmc88q5ft3lm00q30qkd9x6654h3lejnct"
current_dir=$(pwd)
TMP_ROOT="$(dirname $(pwd))/_build"
mkdir -p "${TMP_ROOT}"

TMP_BUILD="${TMP_ROOT}"/jprovd

bypass_go_version_check () {
    VER=$(go version)
    sed -i -e 's/! go version | grep -q "go1.2[0-9]"/false/' ${1}/Makefile
}

install_old () {
    TMP_GOCACHE="${TMP_ROOT}"/gocache
    mkdir -p "${TMP_GOCACHE}"

    echo "installing v3 canined"
    if [[ ! -e "${TMP_ROOT}/canine-chain" ]]; then
        cd ${TMP_ROOT}
        git clone https://github.com/JackalLabs/canine-chain.git
        cd ${current_dir}
    fi

    PROJ_DIR="${TMP_ROOT}/canine-chain"


    cd ${PROJ_DIR}
    git switch tags/${OLD_CHAIN_VER} --detach
    bypass_go_version_check ${PROJ_DIR}
    make install
    git restore Makefile
    echo "finished chain installation"
    
    cd "${current_dir}"
    canined version

    
    if [[ ! -e "${TMP_ROOT}/canine-provider" ]]; then
        cd ${TMP_ROOT}
        git clone https://github.com/JackalLabs/canine-provider.git
        cd canine-provider
        cd ${current_dir}
    fi

    PROJ_DIR="${TMP_ROOT}/canine-provider"


    cd ${PROJ_DIR}
    git switch tags/${OLD_PROVIDER_VER} --detach
    bypass_go_version_check ${PROJ_DIR}
    make install
    git restore Makefile

    cd "${current_dir}"
    jprovd version
}

install_halt () {
    TMP_GOCACHE="${TMP_ROOT}"/gocache
    mkdir -p "${TMP_GOCACHE}"

    echo "installing v3.4 canined"
    if [[ ! -e "${TMP_ROOT}/canine-chain" ]]; then
        cd ${TMP_ROOT}
        git clone https://github.com/JackalLabs/canine-chain.git
        cd ${current_dir}
    fi

    PROJ_DIR="${TMP_ROOT}/canine-chain"


    cd ${PROJ_DIR}
    git switch tags/${HALT_CHAIN_VER} --detach
    bypass_go_version_check ${PROJ_DIR}
    make install
    git restore Makefile
    echo "finished chain installation"

    cd "${current_dir}"
    canined version


    if [[ ! -e "${TMP_ROOT}/canine-provider" ]]; then
        cd ${TMP_ROOT}
        git clone https://github.com/JackalLabs/canine-provider.git
        cd canine-provider
        cd ${current_dir}
    fi

    PROJ_DIR="${TMP_ROOT}/canine-provider"


    cd ${PROJ_DIR}
    git switch tags/${OLD_PROVIDER_VER} --detach
    bypass_go_version_check ${PROJ_DIR}
    make install
    git restore Makefile

    cd "${current_dir}"
    jprovd version
}


install_new () {
    make install
    jprovd version
}

install_new_chain () {
    PROJ_DIR="${TMP_ROOT}/canine-chain"
    cd ${PROJ_DIR}
    #git switch tags/${NEW_CHAIN_VER} --detach
    git switch marston/econ-handler --detach

    bypass_go_version_check ${PROJ_DIR}
    make install
    git restore Makefile

    cd "${current_dir}"
    canined version
}

start_chain () {
    startup
    from_scratch
    fix_config

    screen -d -m -S "canined" bash -c "canined start --pruning=nothing --minimum-gas-prices=0ujkl"
}

set_upgrade_prop () {
    canined tx gov submit-proposal software-upgrade "v4" --upgrade-height ${UPGRADE_HEIGHT} --upgrade-info "tmp" --title "v4 Upgrade" \
        --description "upgrade" --from charlie -y --deposit "20000000ujkl"

    sleep 6

    canined tx gov vote 2 yes --from ${KEY} -y
    echo "voting successful"
}

set_halt_upgrade_prop () {
    canined tx gov submit-proposal software-upgrade "v340" --upgrade-height ${HALT_UPGRADE_HEIGHT} --upgrade-info "tmp" --title "v3 halt Upgrade" \
        --description "upgrade" --from charlie -y --deposit "20000000ujkl"

    sleep 6

    canined tx gov vote 1 yes --from ${KEY} -y
    echo "voting successful"
}

upgrade_chain () {
    while true; do
        BLOCK_HEIGHT=$(canined status | jq '.SyncInfo.latest_block_height' -r)
        if [ $BLOCK_HEIGHT = "$UPGRADE_HEIGHT" ||  $BLOCK_HEIGHT = "$HALT_UPGRADE_HEIGHT"  ]; then
            # assuming running only 1 canined
            echo "BLOCK HEIGHT = $UPGRADE_HEIGHT REACHED, KILLING OLD ONE"
            killall canined
            break
        else
            canined q storage list-active-deals --chain-id test --home $HOME
            canined q storage list-strays --chain-id test --home $HOME
            canined q storage list-contracts --chain-id test --home $HOME
            canined q gov proposal $number --output=json
            echo "BLOCK_HEIGHT = $BLOCK_HEIGHT"
            sleep 2
        fi
    done

    install_new_chain
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

init_sequoia () {
    rm -rf $HOME/providers/sequoia${1}
    sequoia init --home="$HOME/providers/sequoia${1}"

    sed -i -e 's/rpc_addr: https:\/\/jackal-testnet-rpc.polkachu.com:443/rpc_addr: tcp:\/\/localhost:26657/g' $HOME/providers/sequoia${1}/config.yaml
    sed -i -e 's/grpc_addr: jackal-testnet-grpc.polkachu.com:17590/grpc_addr: localhost:9090/g' $HOME/providers/sequoia${1}/config.yaml

    sed -i -e 's/data_directory: $HOME\/.sequoia\/data/data_directory: $HOME\/providers\/sequoia0\/data/g' $HOME/providers/sequoia${1}/config.yaml
}

move_files () {
    cp -rv "${HOME}/providers/provider${1}/ipfs-storage" "${HOME}/providers/sequoia${1}/data"
    cp "${HOME}/providers/provider${1}/config/priv_storkey.json" "${HOME}/providers/sequoia${1}"
}

migrate_sequoia () {
    jprovd migrate-sequoia --home="$HOME/providers/provider$1" -y
    init_sequoia ${1}

    move_files ${1}
}

start_sequoia () {
    sequoia start --home="$HOME/providers/sequoia${1}"
}


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
    killall canined jprovd sequoia
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
sleep 35

# upload files
canined tx storage buy-storage jkl10k05lmc88q5ft3lm00q30qkd9x6654h3lejnct 720h 3000000000 ujkl --from charlie -y
sleep 5

set_halt_upgrade_prop
sleep 5
canined q gov proposal 1

set_upgrade_prop
sleep 5
canined q gov proposal 2

upload_file ./scripts/dummy_data/1.png 0
upload_file ./scripts/dummy_data/2.png 0
#upload_file ./scripts/dummy_data/3.png 0
#upload_file ./scripts/dummy_data/4.png 0
#upload_file ./scripts/dummy_data/5.svg 0
#upload_file ./scripts/dummy_data/6.wav 0
upload_file ./scripts/dummy_data/test.txt 0

sleep 10

upgrade_chain
restart_chain
sleep 10


echo "shutting down providers..."
killall jprovd

sleep 4

echo "upgrading provider..."
install_new

#read -rsp $'Press any key to shutdown and upgrade provider...\n' -n1 key
migrate_provider 0
#migrate_provider 1
#migrate_provider 2
#migrate_provider 3
#migrate_provider 4
#migrate_provider 5
#migrate_provider 6
#
#
#sleep 3
#
read -rsp $'jprov migrate-sequoia now and press any key to continue...\n' -n1 key
# uncomment below to migrate
migrate_sequoia 0

#restart_provider 0
#restart_provider 1
#restart_provider 2
#restart_provider 3
#restart_provider 4
#restart_provider 5
#restart_provider 6



echo "upgrading chain to v4"
upgrade_chain
restart_chain
sleep 5

start_sequoia 0


read -rsp $'Press any key to shutdown...\n' -n1 key

shut_down
#cleanup
