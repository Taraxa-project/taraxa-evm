# go-taraxa-abi

#### Prerequisites
##### solc
sudo add-apt-repository ppa:ethereum/ethereum
sudo apt-get update
sudo apt-get install solc

##### abigen (needed only if implementing client)
go get -u github.com/ethereum/go-ethereum
cd $GOPATH/pkg/mod/github.com/ethereum/go-ethereum/
make
make devtools


#### Create SC ABI
run
```
solc --abi --overwrite --optimize dpos_contract_interface.sol --output-dir .
```

#### Create SC go class
run
```
abigen --abi=DposInterface.abi --pkg=taraxaDposClient --out=dpos_contract_interface.go
```