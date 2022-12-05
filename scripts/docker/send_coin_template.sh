#!/bin/bash
#you should update the template shell your self
#"omni_send" mtowceAw2yeftR1pPg15QcsDqsnSik7Spz "$1"  $2 100; more doc: https://github.com/OmniLayer/omnicore/blob/master/src/omnicore/doc/rpc-api.md#omni_send
# docker-image example ccr.ccs.tencentyun.com/omnicore/omnicored-noinit:0.0.1
docker  exec -u 1000 omnode1  omnicore-cli -testnet "omni_send" "mtowceAw2yeftR1pPg15QcsDqsnSik7Spz" "$1" $2 100
echo asset balance:
docker  exec -u 1000 omnode1  omnicore-cli -testnet omni_getbalance  $1 $2