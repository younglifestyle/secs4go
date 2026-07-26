package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/parser/hsms"
)

var ErrMsgParsing = errors.New("message parsing error")
var ErrMsgFormat = errors.New("message format error")
var ErrMsgTooLarge = errors.New("message exceeds configured maximum")

const DefaultMaxSECSMessageSize uint32 = 16 * 1024 * 1024

type SecsIIProtocol struct {
	maxMessageSize uint32
}

func (s *SecsIIProtocol) NewCodec(rw io.ReadWriter) (link.Codec, error) {
	codec := &secsIICodec{
		rw:             rw,
		headBuf:        make([]byte, 4),
		maxMessageSize: s.maxMessageSize,
		headDecoder: func(bytes []byte) uint32 {
			return binary.BigEndian.Uint32(bytes)
		},
	}
	codec.closer, _ = rw.(io.Closer)
	return codec, nil
}

func SECSII() *SecsIIProtocol {
	return &SecsIIProtocol{maxMessageSize: DefaultMaxSECSMessageSize}
}

// SECSIIWithMaxMessageSize creates a codec with an explicit receive limit.
// A zero limit uses DefaultMaxSECSMessageSize.
func SECSIIWithMaxMessageSize(max uint32) *SecsIIProtocol {
	if max == 0 {
		max = DefaultMaxSECSMessageSize
	}
	return &SecsIIProtocol{maxMessageSize: max}
}

// MessageParseError retains the offending header so the HSMS layer can send S9F7.
type MessageParseError struct {
	Header []byte
}

func (e *MessageParseError) Error() string  { return ErrMsgParsing.Error() }
func (e *MessageParseError) Unwrap() error  { return ErrMsgParsing }
func (e *MessageParseError) NonFatal() bool { return true }

// MessageTooLargeError retains the available header so the HSMS layer can send S9F11.
type MessageTooLargeError struct {
	Length uint32
	Header []byte
}

func (e *MessageTooLargeError) Error() string {
	return fmt.Sprintf("%v: %d bytes", ErrMsgTooLarge, e.Length)
}
func (e *MessageTooLargeError) Unwrap() error  { return ErrMsgTooLarge }
func (e *MessageTooLargeError) NonFatal() bool { return true }

type secsIICodec struct {
	rw                    io.ReadWriter
	closer                io.Closer
	headBuf               []byte
	bodyBuf               []byte
	headDecoder           func([]byte) uint32
	headEncoder           func([]byte, uint32)
	maxMessageSize        uint32
	intercharacterTimeout time.Duration
}

func (c *secsIICodec) Receive() (interface{}, error) {
	// Waiting for the first byte is not governed by T8. Once a message starts,
	// the deadline is reset for each subsequent read.
	if _, err := io.ReadFull(c.rw, c.headBuf[:1]); err != nil {
		return nil, err
	}
	if err := c.readIntercharacter(c.headBuf[1:]); err != nil {
		return nil, err
	}

	msgLength := c.headDecoder(c.headBuf)
	if msgLength < 10 {
		return nil, ErrMsgFormat
	}
	if c.maxMessageSize == 0 {
		c.maxMessageSize = DefaultMaxSECSMessageSize
	}
	if msgLength > c.maxMessageSize {
		header := make([]byte, 10)
		if err := c.readIntercharacter(header); err != nil {
			return nil, err
		}
		return nil, &MessageTooLargeError{Length: msgLength, Header: header}
	}

	required := int(msgLength) + 4
	if cap(c.bodyBuf) < required {
		c.bodyBuf = make([]byte, required)
	} else {
		c.bodyBuf = c.bodyBuf[:required]
	}

	binary.BigEndian.PutUint32(c.bodyBuf, msgLength)

	if err := c.readIntercharacter(c.bodyBuf[4:required]); err != nil {
		return nil, err
	}

	msg, ok := hsms.Parse(c.bodyBuf[:required])
	if !ok {
		header := make([]byte, 10)
		copy(header, c.bodyBuf[4:14])
		return nil, &MessageParseError{Header: header}
	}

	return msg, nil
}

func (c *secsIICodec) SetIntercharacterTimeout(timeout time.Duration) {
	c.intercharacterTimeout = timeout
}

type readDeadlineSetter interface {
	SetReadDeadline(time.Time) error
}

func (c *secsIICodec) readIntercharacter(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	setter, supportsDeadline := c.rw.(readDeadlineSetter)
	if !supportsDeadline || c.intercharacterTimeout <= 0 {
		_, err := io.ReadFull(c.rw, buf)
		return err
	}

	defer setter.SetReadDeadline(time.Time{})
	for offset := 0; offset < len(buf); {
		if err := setter.SetReadDeadline(time.Now().Add(c.intercharacterTimeout)); err != nil {
			return err
		}
		n, err := c.rw.Read(buf[offset:])
		offset += n
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func (c *secsIICodec) Send(msg interface{}) error {
	b, ok := msg.([]byte)
	if !ok {
		return ErrMsgFormat
	}
	for len(b) > 0 {
		n, err := c.rw.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}

func (c *secsIICodec) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
