package myrpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

type Bar int

func (b Bar) Timeout(argv int, reply *int) error {
	time.Sleep(time.Second * 2)
	return nil
}

func TestClient_dialTimeout(t *testing.T) {
	t.Parallel()
	lis, _ := net.Listen("tcp", ":0")
	f := func(conn net.Conn, opt *Option) (*Client, error) {
		_ = conn.Close()
		time.Sleep(time.Second * 2)
		return nil, nil
	}
	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", lis.Addr().String(), &Option{ConnectTimeout: time.Second})
		fmt.Println(err)
		_assert(err != nil &&  strings.Contains(err.Error(), "connect timeout"), "expect a timeout error")
	})
	t.Run("0", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", lis.Addr().String(), &Option{ConnectTimeout: 0})
		fmt.Println(err)
		_assert(err == nil, "0 means no limit")
	})
}

func startServer(addr chan string) {
	var b Bar
	_ = Register(&b)
	lis, _ := net.Listen("tcp", ":0")
	addr <- lis.Addr().String()
	Accept(lis)
}

func TestClient_Call(t *testing.T) {
	t.Parallel()
	addr := make(chan string)
	go startServer(addr)

	addresss := <-addr
	// 验证Call超时
	// 设置一个call超时时间，会把消息发送给服务器，然后等待服务器处理并返回结果。
	// 若在规定的call超时时间内，没有把消息发送给服务器或服务器也没有在规定时间内返回结果，那么判定为超时
	// 这时候服务器仍然在后台处理数据并返回，但是这个结果不会赋值给对应的call。而是通过nil读取结果
	t.Run("client call timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addresss)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		var reply int
		err := client.Call(ctx, "Bar.Timeout", 1, &reply)
		fmt.Println(err, "###", reply)
		_assert(err != nil && strings.Contains(err.Error(), ctx.Err().Error()), "expect a timeout error")
	})
	// 验证服务器处理超时
	t.Run("client handle timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addresss, &Option{
			HandleTimeout: time.Second,
		})
		var reply int
		err := client.Call(context.Background(), "Bar.Timeout", 1, &reply)
		fmt.Println(err, "##@@@###", reply)
		_assert(err != nil && strings.Contains(err.Error(), "handle timeout"), "expect a timeout error")
	})
}