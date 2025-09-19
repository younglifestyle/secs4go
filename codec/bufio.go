package codec

import (
	"bufio"
	"io"
	"time"

	link "github.com/younglifestyle/secs4go"
)

type deadlineConn interface {
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

func Bufio(base link.Protocol, readBuf, writeBuf int) link.Protocol {
	return &bufioProtocol{
		base:     base,
		readBuf:  readBuf,
		writeBuf: writeBuf,
	}
}

type bufioProtocol struct {
	base     link.Protocol
	readBuf  int
	writeBuf int
}

func (b *bufioProtocol) NewCodec(rw io.ReadWriter) (cc link.Codec, err error) {
	codec := new(bufioCodec)

	if b.writeBuf > 0 {
		codec.stream.w = bufio.NewWriterSize(rw, b.writeBuf)
		codec.stream.Writer = codec.stream.w
	} else {
		codec.stream.Writer = rw
	}

	if b.readBuf > 0 {
		codec.stream.Reader = bufio.NewReaderSize(rw, b.readBuf)
	} else {
		codec.stream.Reader = rw
	}

	codec.stream.c, _ = rw.(io.Closer)
	if dc, ok := rw.(deadlineConn); ok {
		codec.stream.deadline = dc
	}

	codec.base, err = b.base.NewCodec(&codec.stream)
	if err != nil {
		return
	}
	cc = codec
	return
}

type bufioStream struct {
	io.Reader
	io.Writer
	c        io.Closer
	w        *bufio.Writer
	deadline deadlineConn
}

func (s *bufioStream) Flush() error {
	if s.w != nil {
		return s.w.Flush()
	}
	return nil
}

func (s *bufioStream) close() error {
	if s.c != nil {
		return s.c.Close()
	}
	return nil
}

func (s *bufioStream) SetReadDeadline(t time.Time) error {
	if s.deadline != nil {
		return s.deadline.SetReadDeadline(t)
	}
	return nil
}

func (s *bufioStream) SetWriteDeadline(t time.Time) error {
	if s.deadline != nil {
		return s.deadline.SetWriteDeadline(t)
	}
	return nil
}

func (s *bufioStream) SetDeadline(t time.Time) error {
	if s.deadline != nil {
		return s.deadline.SetDeadline(t)
	}
	return nil
}

type bufioCodec struct {
	base   link.Codec
	stream bufioStream
}

func (c *bufioCodec) Send(msg interface{}) error {
	if err := c.base.Send(msg); err != nil {
		return err
	}
	return c.stream.Flush()
}

func (c *bufioCodec) Receive() (interface{}, error) {
	return c.base.Receive()
}

func (c *bufioCodec) Close() error {
	err1 := c.base.Close()
	err2 := c.stream.close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (c *bufioCodec) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c *bufioCodec) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}
