package dlvrpc

import (
	"log"
	"time"

	"github.com/go-delve/delve/service/rpc2"
)

type RPCClient struct {
	dlvclient *rpc2.RPCClient
}

func checkOnline(addr string) RPCClient {
    defer func() {
        if err := recover(); err != nil {
            log.Println("Retrying. Connection failed:", err)
        }
    }()
	return RPCClient{
		dlvclient: rpc2.NewClient(addr),
	}
}

func NewClient(addr string, clientChan chan *rpc2.RPCClient) {

	c1 := make(chan string, 1)
    go func() {
        time.Sleep(1 * time.Second)
        c1 <- "Timeout done"
    }()
	done := <- c1
	log.Print(done)
	clientChan <- rpc2.NewClient(addr)
}

