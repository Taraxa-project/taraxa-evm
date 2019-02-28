package ethdb

import (
	"context"
	"github.com/Taraxa-project/taraxa-evm/grpc/grpc_go"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

type RpcDatabase struct {
	client grpc_go.StateDBClient
}

func NewRpcDatabase(conn *grpc.ClientConn) *RpcDatabase {
	return &RpcDatabase{client: grpc_go.NewStateDBClient(conn)}
}

func (rpcDatabase *RpcDatabase) Put(key []byte, value []byte) error {
	_, err := rpcDatabase.client.Put(context.Background(), &grpc_go.BytesMessage{Value: key})
	return err
}

func (rpcDatabase *RpcDatabase) Delete(key []byte) error {
	_, err := rpcDatabase.client.Delete(context.Background(), &grpc_go.BytesMessage{Value: key})
	return err
}

func (rpcDatabase *RpcDatabase) Get(key []byte) ([]byte, error) {
	msg, err := rpcDatabase.client.Get(context.Background(), &grpc_go.BytesMessage{Value: key})
	return msg.Value, err
}

func (rpcDatabase *RpcDatabase) Has(key []byte) (bool, error) {
	msg, err := rpcDatabase.client.Has(context.Background(), &grpc_go.BytesMessage{Value: key})
	return msg.Value, err
}

func (rpcDatabase *RpcDatabase) Close() {
	_, err := rpcDatabase.client.Close(context.Background(), &empty.Empty{})
	if err != nil {
		panic(err)
	}
}

func (rpcDatabase *RpcDatabase) NewBatch() Batch {
	return &MemBatch{
		db: rpcDatabase,
	}
}
