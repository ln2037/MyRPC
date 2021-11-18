package main

import (
	"MyRpc/02_client/myrpc"
	"fmt"
	"log"
	"net"
	"sync"
)

func startServer(addr chan string) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", lis.Addr())
	addr <- lis.Addr().String()
	myrpc.Accept(lis)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	address := <- addr
	client, _ := myrpc.Dial("tcp", address)
	defer func() {client.Close()}()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("args", args, "reply: ", reply)
			fmt.Println("req", i, "replay:", reply)
		}(i)
	}
	wg.Wait()
}
