module github.com/Taraxa-project/taraxa-evm

go 1.13

require (
	github.com/aristanetworks/goarista v0.0.0-20200310212843-2da4c1f5881b
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/davecgh/go-spew v1.1.1
	github.com/emicklei/dot v0.10.2
	github.com/emirpasic/gods v1.12.0
	github.com/fjl/gencodec v0.0.0-20191126094850-e283372f291f // indirect
	github.com/go-stack/stack v1.8.0
	github.com/google/uuid v1.1.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/stretchr/testify v1.5.1
	github.com/tecbot/gorocksdb v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
)

replace github.com/tecbot/gorocksdb => github.com/02p01r/gorocksdb v0.0.0-20200326074958-c63f8b69db1e
