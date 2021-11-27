package codec

import "io"

type Header struct {
	ServiceMethod	string // format "Service.Method"
	Seq 			uint64 // sequence number chosen by client
	Error 			string
}

// Codec 对消息体进行编码解码的接口。抽象出此接口是为了实现不同的codec实例
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// 构造函数
type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}