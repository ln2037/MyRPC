package myrpc

import (
	"MyRpc/02_client/myrpc/codec"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

// Option 编码方式
type Option struct {
	MagicNumber	int			// 标记这是myrpc请求
	CodecType	codec.Type	// 编码类型
}

var DefaultOption = &Option{
	MagicNumber:	MagicNumber,
	CodecType:		codec.GobType,
}

// Server 代表一个MyRpc服务
type Server struct { }

// NewServer 返回一个MyRpc实例
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

// ServeConn runs the server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		// 退出后关闭
		conn.Close()
	}()
	// 获取编码方式
	var option Option
	if err := json.NewDecoder(conn).Decode(&option); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	//检查MagicNumber和CodeType是否正确
	if option.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}
	newCodec := codec.NewCodecFuncMap[option.CodecType]
	if newCodec == nil {
		log.Printf("rpc server: invalid codec type %s", option.CodecType)
		return
	}
	//获取消息的解码器
	//调用serveCodec
	server.ServeCodec(newCodec(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) ServeCodec(cc codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		// 读取请求
		req, err := server.readRequest(cc)
		if err != nil {
			// req为nil，直接结束
			if req == nil {
				break
			}
			// 返回错误信息
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		//请求无误，开始处理
		go server.handleRequest(cc, req, sending, wg)
	}
	//等待所有请求处理完毕
	wg.Wait()
	cc.Close()
}

// 对请求进行处理
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.header, req.argv.Elem())
	// 返回结果
	req.replyv = reflect.ValueOf(fmt.Sprintf("MyRpc resp %d", req.header.Seq))
	server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
}

func (server *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	//fmt.Println("fmt",body)
	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var header codec.Header
	if err := cc.ReadHeader(&header); err != nil {
		// TODO 在哪种情况下返回eof
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &header, nil
}

// request stores all information of a call
type request struct {
	header            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	// 获取请求的Header
	header, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{header: header}
	// 获取请求的Body, 需要反射
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

// Accept accepts connections on the listener and serve request
// for each incoming connection
func (server *Server) Accept(lis net.Listener) {
	for {
		// 开始监听
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		//fmt.Println("server accept")
		go server.ServeConn(conn)
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}
