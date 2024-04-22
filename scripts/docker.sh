cp -r /root/.jackal-storage/config /copyconfig


jprovd init $PROVIDER_DOMAIN $PROVIDER_SPACE "" -y || true

sleep 10

jprovd start -y --moniker=$PROVIDER_NAME || true

/bin/bash