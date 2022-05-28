package dlvrpc

import (
	"fmt"
	"net"

	"github.com/go-delve/delve/service/rpc2"
)

type RPCClient struct {
	server *rpc2.RPCClient
}
