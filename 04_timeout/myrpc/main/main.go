package main

import (
	"MyRpc/04_timeout/myrpc"
	"context"
	"log"
	"net"
	"sync"
)

type Foo int

type Args struct { Num1, Num2 int}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := myrpc.Register(&foo); err != nil {
		log.Fatal("register error: ", err)
	}
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	log.Println("start rpc server on ", lis.Addr())
	addr <- lis.Addr().String()
	myrpc.Accept(lis)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	address := <-addr
	client, _ := myrpc.Dial("tcp", address)
	defer func() {client.Close()}()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			//ctx, _ := context.WithTimeout(context.Background(), time.Second)
			ctx := context.Background()
			if err := client.Call(ctx, "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}
