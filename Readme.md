## omnicored faucet api

This faucet is for omnicore regtest/testnet only.  

### main api

* mine
* send_coin
* get asset balance
* list assets
* query asset
* create asset  

### swagger doc  

https://swagger.oblnd.top/?surl=https://faucet.oblnd.top/openapiv2/foo.swagger.json

![swagger preview](https://raw.githubusercontent.com/omnilaboratory/omnicore-fauct-api/master/swagger/img1.png "swagger image")

### programe start  
```
#omnicore-proxy-server
go run whitelist_proxy --btc_host=xx.xx.xx.xx --rpc_user=xx --rpc_passwd=xxx

#faucet-api: 
go run main.go --btc_host=omnicore-proxy-ip-xx.xx.xx

```

### pre-created regtest net omnicoreporxy
web have a pre-created omniproxy-server for regtest
* server domain: regnet.oblnd.top
* port: --bitcoin.active --bitcoin.regtest --bitcoin.node=omnicoreproxy --omnicoreproxy.rpchost=regnet.oblnd.top:18332 --omnicoreproxy.zmqpubrawblock=tcp://regnet.oblnd.top:28332 --omnicoreproxy.zmqpubrawtx=tcp://regnet.oblnd.top:28333
* omnicoreporxy is public prxoy omnicore-backand ,it can be access anonymous.
* faucet-swager-api: https://swagger.oblnd.top/?surl=https://faucet.oblnd.top/openapiv2/foo.swagger.json
* omnicoreporxy-server wallet addre ms5u6Wmc8xF8wFBo9w5HFouFNAmnWzkVa6 have enough test-coin to send you.
* omnicoreporxy have pre-created an asset which id is 2147483651;
*  the SendCoin api will every invoke will send you 1btc and 100 asset.
```shell
#send test coin form curl
export assetId=2147483651
curl -X 'GET' \
  'https://faucet.oblnd.top/api/SendCoin/$a_address?assetId=$assetId' \
  -H 'accept: application/json'
```
