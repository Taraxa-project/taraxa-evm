# go-taraxa-abi

##### abigen (needed only if implementing client)
go get -u github.com/ethereum/go-ethereum
cd $GOPATH/pkg/mod/github.com/ethereum/go-ethereum/
make
make devtools

#### Create SC go class
run
```
abigen --abi=abi/DposInterface.abi --pkg=taraxaDposClient --out=dpos_contract_interface.go
```