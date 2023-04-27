## omnicored proxy api

This proxy offers public anonymous omnicore services to anonymous users and is currently for regtest/testnet only. It acts as the backend of OBD and can be deployed on a remote cloud. The motivation is to decouple the lightning node and the full Bitcoin/Omnilayer node, to lower the barriers of OBD deployment. 

The mainnet version will be available after omnicore V0.12 is activated.  

### proxy api  
The complete white-listed interfaces are in: [https://github.com/omnilaboratory/omnicore-proxy/blob/master/whitelist_proxy/whitelist_proxy.go](https://github.com/omnilaboratory/omnicore-proxy/blob/master/whitelist_proxy/whitelist_proxy.go)  

### faucet api

* mine: mine blocks, regtest only
* send_coin: the faucet sending tokens to an address, regtest/testnet only
* get asset balance
* list assets
* query asset
* create asset  

### faucet api swagger doc  

https://swagger.oblnd.top/?surl=https://faucet.oblnd.top/openapiv2/foo.swagger.json

![swagger preview](https://raw.githubusercontent.com/omnilaboratory/omnicore-fauct-api/master/swagger/img1.png "swagger image")

### programe start  
```
#omnicore-proxy-server
go run whitelist_proxy --btc_host=xx.xx.xx.xx --rpc_user=xx --rpc_passwd=xxx

#faucet-api: 
go run main.go --btc_host=btc_fullnode_ip-xx.xx.xx

```

### omnicore proxy on Regtest
web have deployed an omniproxy-server on regtest for developers:
* server domain: `regnet.oblnd.top`  
* port: `--bitcoin.active --bitcoin.regtest --bitcoin.node=omnicoreproxy --omnicoreproxy.rpchost=regnet.oblnd.top:18332 --omnicoreproxy.zmqpubrawblock=tcp://regnet.oblnd.top:28332 --omnicoreproxy.zmqpubrawtx=tcp://regnet.oblnd.top:28333`  
* omnicore proxy is a public omnicore-backend, it can be accessed anonymously.  
* faucet-swager-api: `https://swagger.oblnd.top/?surl=https://faucet.oblnd.top/openapiv2/foo.swagger.json`  
* omnicoreproxy-server wallet address: `ms5u6Wmc8xF8wFBo9w5HFouFNAmnWzkVa6` have enough test-coin to send you. 
* we issued an asset with id `2147483651` on the faucet.  
* the SendCoin api will send you 1 btc and 100 tokens(asset ID = 2147483651).  

```shell
#send test coin form curl
export assetId=2147483651
curl -X 'GET' \
  'https://faucet.oblnd.top/api/SendCoin/$a_address?assetId=$assetId' \
  -H 'accept: application/json'
```

### agency fund api (spay)
luckpackage and agency fund service
The complete interfaces are in: toolrpc/luckPk.proto

go run spay [falgs]

