# go-taraxa-abi
Read tutorial inside slashing_contract_solidity_structs.go file

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
solc --abi --overwrite --optimize slashing_contract_interface.sol --output-dir .
```

#### Create SC go class
run
```
abigen --abi=SlashingInterface.abi --pkg=taraxaSlashingClient --out=slashing_contract_interface.go
```