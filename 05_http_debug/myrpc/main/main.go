package main

import (
	"MyRpc/05_http_debug/myrpc"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
)

type Foo int

type Args struct { Num1, Num2 int}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addrCh chan string) {
	var foo Foo
	lis, _ := net.Listen("tcp", ":9999")
	myrpc.Register(&foo)
	// 注册Handler
	myrpc.HandleHTTP()
	addrCh <-lis.Addr().String()
	_ = http.Serve(lis, nil)
}

func call(addrCh chan string) {
	address := <-addrCh
	fmt.Println(address)
	client, _ := myrpc.DialHTTP("tcp", address)
	defer func() { _ = client.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	ch := make(chan string)
	go call(ch)
	startServer(ch)
}
