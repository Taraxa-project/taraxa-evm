package ethdb

import (
	"context"
	"github.com/Taraxa-project/taraxa-evm/grpc/grpc_go"
	"google.golang.org/grpc"
)

type VmId = grpc_go.VmId

type RpcDatabase struct {
	client grpc_go.StateDBClient
	vmId   *VmId
}

func NewRpcDatabase(conn *grpc.ClientConn, vmId *VmId) *RpcDatabase {
	return &RpcDatabase{
		client: grpc_go.NewStateDBClient(conn),
		vmId:   vmId,
	}
}

func (rpcDatabase *RpcDatabase) Put(key []byte, value []byte) error {
	_, err := rpcDatabase.client.Put(context.Background(), &grpc_go.KeyAndValueMessage{
		Key: &grpc_go.KeyMessage{
			VmId:          rpcDatabase.vmId,
			MemoryAddress: &grpc_go.BytesMessage{Value: key},
		},
		Value: &grpc_go.BytesMessage{Value: value},
	})
	return err
}

func (rpcDatabase *RpcDatabase) Delete(key []byte) error {
	_, err := rpcDatabase.client.Delete(context.Background(), &grpc_go.KeyMessage{
		VmId:          rpcDatabase.vmId,
		MemoryAddress: &grpc_go.BytesMessage{Value: key},
	})
	return err
}

func (rpcDatabase *RpcDatabase) Get(key []byte) ([]byte, error) {
	msg, err := rpcDatabase.client.Get(context.Background(), &grpc_go.KeyMessage{
		VmId:          rpcDatabase.vmId,
		MemoryAddress: &grpc_go.BytesMessage{Value: key},
	})
	return msg.Value, err
}

func (rpcDatabase *RpcDatabase) Has(key []byte) (bool, error) {
	msg, err := rpcDatabase.client.Has(context.Background(), &grpc_go.KeyMessage{
		VmId:          rpcDatabase.vmId,
		MemoryAddress: &grpc_go.BytesMessage{Value: key},
	})
	return msg.Value, err
}

func (rpcDatabase *RpcDatabase) Close() {
	_, err := rpcDatabase.client.Close(context.Background(), rpcDatabase.vmId)
	if err != nil {
		// the original interface doesn't support errors
		panic(err)
	}
}

func (rpcDatabase *RpcDatabase) NewBatch() Batch {
	return &MemBatch{
		putter:  rpcDatabase,
		deleter: rpcDatabase,
	}
}
