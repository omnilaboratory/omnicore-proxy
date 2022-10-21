## omnicored proxy api

This proxy offers anonymous omnicore services to anonymous users and is currently for regtest/testnet only. The mainnet version will be available after omnicore V0.12 is activated.   

### main api

* mine: mine blocks, regtest only
* send_coin: the faucet sending tokens to an address, regtest/testnet only
* get_asset_balance
* list_assets
* query_asset
* create_asset  

### swagger doc  

http://62.234.169.68:29082/swagger/?surl=http://43.138.107.248:8090/openapiv2/foo.swagger.json  

![swagger preview](https://raw.githubusercontent.com/omnilaboratory/omnicore-fauct-api/master/swagger/img.png "swagger image")  


### programe start  
```
go run main.go
```
