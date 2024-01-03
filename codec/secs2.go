package codec

import (
	"encoding/binary"
	"errors"
	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/parser/hsms"
	"io"
)

var ErrMsgParsing = errors.New("message parsing error")
var ErrMsgFormat = errors.New("message format error")

type SecsIIProtocol struct {
}

func (b *SecsIIProtocol) NewCodec(rw io.ReadWriter) (link.Codec, error) {
	codec := &secsIICodec{
		rw:      rw,
		headBuf: make([]byte, 4), // 长度
		headDecoder: func(bytes []byte) uint32 {
			return binary.BigEndian.Uint32(bytes)
		},
	}

	codec.closer, _ = rw.(io.Closer)
	return codec, nil
}

func SECSII() *SecsIIProtocol {
	return &SecsIIProtocol{}
}

type secsIICodec struct {
	rw          io.ReadWriter
	closer      io.Closer
	headBuf     []byte
	bodyBuf     []byte
	headDecoder func([]byte) uint32
	headEncoder func([]byte, uint32)
}

func (c *secsIICodec) Receive() (interface{}, error) {
	var msgLength uint32

	// windows读取0也会返回io.EOF错误，与链接关闭的返回是一样的，无法区分
	// 查看源码net.go，这一块没办法
	// https://github.com/golang/go/issues/15735，已在1.7版本解决
	// 再次勘误，查出原因是因为对端也传递了linktest.req请求，最初未处理，导致对端将连接close了
	_, err := c.rw.Read(c.headBuf)
	if err != nil {
		return nil, err
	}
	msgLength = c.headDecoder(c.headBuf)

	// 协议允许的数据包很大，无需特殊考虑最大包大小
	if msgLength < 10 {
		return nil, ErrMsgFormat
	}
	if cap(c.bodyBuf) < int(msgLength)+4 {
		c.bodyBuf = make([]byte, msgLength+4)
	}

	_, err = c.rw.Read(c.bodyBuf[4 : msgLength+4])
	if err != nil {
		return nil, err
	}
	// 重组消息数据
	binary.BigEndian.PutUint32(c.bodyBuf, msgLength)
	msg, ok := hsms.Parse(c.bodyBuf[0 : msgLength+4])
	if !ok {
		return nil, ErrMsgParsing
	}

	return msg, nil
}

func (c *secsIICodec) Send(msg interface{}) error {

	b, ok := msg.([]byte)
	if !ok {
		return ErrMsgFormat
	}

	_, err := c.rw.Write(b)

	return err
}

func (c *secsIICodec) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
