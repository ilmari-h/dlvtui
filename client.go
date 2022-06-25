package main

import (
	"net"
	"time"

	"github.com/go-delve/delve/service/rpc2"
	log "github.com/sirupsen/logrus"
)

func NewClient(addr string, clientChan chan *rpc2.RPCClient) {

	attempts := 0
	for {
		conn, _ := net.Dial("tcp", addr)
		time.Sleep(time.Second / 10)
		if conn != nil {
			log.Print("Client connection established.")
			clientChan <- rpc2.NewClientFromConn(conn)
			break
		}
		attempts++
		if attempts >= 50 {
			log.Fatalf("Failed to connect client to dlv backend after %d attempts!", attempts)
		}
		log.Printf("Client connection to %s refused, retry number %d.", addr, attempts)
	}
}
