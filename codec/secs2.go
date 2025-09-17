package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/parser/hsms"
)

var ErrMsgParsing = errors.New("message parsing error")
var ErrMsgFormat = errors.New("message format error")

type SecsIIProtocol struct{}

func (s *SecsIIProtocol) NewCodec(rw io.ReadWriter) (link.Codec, error) {
	codec := &secsIICodec{
		rw:      rw,
		headBuf: make([]byte, 4),
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
	if _, err := io.ReadFull(c.rw, c.headBuf); err != nil {
		return nil, err
	}

	msgLength := c.headDecoder(c.headBuf)
	if msgLength < 10 {
		return nil, ErrMsgFormat
	}

	required := int(msgLength) + 4
	if cap(c.bodyBuf) < required {
		c.bodyBuf = make([]byte, required)
	} else {
		c.bodyBuf = c.bodyBuf[:required]
	}

	binary.BigEndian.PutUint32(c.bodyBuf, msgLength)

	if _, err := io.ReadFull(c.rw, c.bodyBuf[4:required]); err != nil {
		return nil, err
	}

	msg, ok := hsms.Parse(c.bodyBuf[:required])
	if !ok {
		fmt.Printf("hsms decode failed, len=%d data=%x\n", required, c.bodyBuf[:required])
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
