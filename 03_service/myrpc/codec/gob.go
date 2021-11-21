package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec 实现了Codec接口
type GobCodec struct {
	conn	io.ReadWriteCloser
	buf		*bufio.Writer
	encoder	*gob.Encoder
	decoder	*gob.Decoder
}

var _ Codec = (*GobCodec)(nil)

// NewGobCodec 返回gob编码解码器的实例
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	//使用bufio能够提高效率
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: 		conn,
		buf:		buf,
		encoder:	gob.NewEncoder(buf),
		decoder:	gob.NewDecoder(conn),
	}
}

// ReadHeader 获取Header
func (c *GobCodec) ReadHeader(header *Header) error {
	return c.decoder.Decode(header)
}

// ReadBody 读取body的数据
func (c *GobCodec) ReadBody(body interface{}) error {
	// 传递的是一个指针
	return c.decoder.Decode(body)
}

// Write 向连接中写入数据。使用bufio来提高效率
func (c *GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		//把写入的内容刷出缓冲区
		_ = c.buf.Flush()
		// 若出错，关闭连接
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.encoder.Encode(header); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return
	}
	//fmt.Println("fmt2", body)
	if err = c.encoder.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return
	}
	return
}

// Close 关闭连接
func (c *GobCodec) Close() error {
	return c.conn.Close()
}