package main

import (
	"MyRpc/01_codec/myrpc"
	"MyRpc/01_codec/myrpc/codec"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

// 开启服务
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

	// 等待服务器开启后再新建连接
	address := <- addr
	conn, _ := net.Dial("tcp", address)
	defer func() { conn.Close() }()

	time.Sleep(time.Second)

	// 向服务器发送编码方式
	option := myrpc.DefaultOption
	json.NewEncoder(conn).Encode(option)
	cc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		header := &codec.Header{
			ServiceMethod: 	"Foo.Sum",
			Seq:			uint64(i),
		}
		cc.Write(header, fmt.Sprintf("MyRpc req %d", header.Seq))
		cc.ReadHeader(header)
		//fmt.Println(header)
		var reply string
		cc.ReadBody(&reply)
		log.Println("reply: ", reply)
	}
}