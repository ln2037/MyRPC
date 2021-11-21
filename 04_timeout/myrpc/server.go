package myrpc

import (
	"MyRpc/04_timeout/myrpc/codec"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNumber = 0x3bef5c

// Option 编码方式
type Option struct {
	MagicNumber		int			// 标记这是myrpc请求
	CodecType		codec.Type	// 编码类型
	ConnectTimeout	time.Duration
	HandleTimeout	time.Duration
}

var DefaultOption = &Option{
	MagicNumber:	MagicNumber,
	CodecType:		codec.GobType,
	ConnectTimeout:	time.Second * 10,
}

// Server 代表一个MyRpc服务
type Server struct {
	serviceMap sync.Map
}

// NewServer 返回一个MyRpc实例
func NewServer() *Server {
	return &Server{}
}

// 找到对应的服务和方法
func (server *Server) findService(serviceMethod string) (svc *service, metType *methodType, err error) {
	// 找出服务名和方法名
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[: dot], serviceMethod[dot + 1: ]
	serv, ok := server.serviceMap.Load(serviceName)
	// 若服务未注册，返回
	if ok == false {
		err = errors.New("rpc server: can`t find service " + serviceName)
		return
	}
	svc = serv.(*service)
	metType = svc.method[methodName]
	// 若方法不存在，返回
	if metType == nil {
		err = errors.New("rpc server: can`t find method " + methodName)
	}
	return
}

// 发布方法
func (server *Server) Register(receiver interface{}) error {
	// 获得一个服务
	s := newService(receiver)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

// 发布receiver的方法
func Register(receiver interface{}) error {
	return DefaultServer.Register(receiver)
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
	server.ServeCodec(newCodec(conn), &option)
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) ServeCodec(cc codec.Codec, opt *Option) {
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
		go server.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
	}
	//等待所有请求处理完毕
	wg.Wait()
	cc.Close()
}

// 对请求进行处理
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.service.call(req.metType, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			// 这里不能忘
			sent <- struct{}{}
			return
		}
		// 返回结果
		server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()
	// 若没有设置超时
	if timeout == 0 {
		<-called
		<-sent
		// 直接返回
		return
	}
	// 设置了超时时间
	select {
	case <-time.After(timeout):
		// 本来想超时后会不会这里设置header的error后，已经开启的go程把它修改了。后来发现不会
		// 在超时前的代码里，不会修改header.error的值
		req.header.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(cc, req.header, invalidRequest, sending)
	case <-called:
		<-sent
	}
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
	header			*codec.Header // header of request
	argv, replyv 	reflect.Value // argv and replyv of request
	metType			*methodType	// 方法类型
	service			*service // 服务
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	// 获取请求的Header
	header, err := server.readRequestHeader(cc)
	if err != nil 	{
		return nil, err
	}
	req := &request{header: header}
	req.service, req.metType, err = server.findService(req.header.ServiceMethod)
	if err != nil {
		return nil, err
	}
	// 获取到的是一个实例化的值,并能够修改其值
	req.argv = req.metType.newArgv()
	req.replyv = req.metType.newReplyv()

	//需要获得req.argv的指针，这样能通过readBody为其赋值
	argvi := req.argv.Interface()
	if req.argv.Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	// 获取请求的Body。argvi是一个指针类型
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
		return req, err
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
