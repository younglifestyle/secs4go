package codec

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func TestSecsIICodecRejectsOversizedMessageBeforeAllocation(t *testing.T) {
	raw := append([]byte{0, 0, 0, 11}, make([]byte, 10)...)
	codec, err := SECSIIWithMaxMessageSize(10).NewCodec(bytes.NewBuffer(raw))
	if err != nil {
		t.Fatal(err)
	}
	_, err = codec.Receive()
	var tooLarge *MessageTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected MessageTooLargeError, got %v", err)
	}
	if tooLarge.Length != 11 || len(tooLarge.Header) != 10 {
		t.Fatalf("unexpected error details: %#v", tooLarge)
	}
}

func TestSecsIIT8StartsAfterFirstByte(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	codec, err := SECSII().NewCodec(client)
	if err != nil {
		t.Fatal(err)
	}
	codec.(interface{ SetIntercharacterTimeout(time.Duration) }).SetIntercharacterTimeout(20 * time.Millisecond)

	result := make(chan error, 1)
	go func() {
		_, receiveErr := codec.Receive()
		result <- receiveErr
	}()

	// An idle connection must not trigger T8 before a message starts.
	time.Sleep(40 * time.Millisecond)
	raw := ast.NewHSMSMessageLinktestReq([]byte{0, 0, 0, 1}).ToBytes()
	if _, err := server.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil {
		t.Fatalf("idle connection incorrectly triggered T8: %v", err)
	}
}

func TestSecsIIT8ExpiresBetweenCharacters(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	codec, err := SECSII().NewCodec(client)
	if err != nil {
		t.Fatal(err)
	}
	codec.(interface{ SetIntercharacterTimeout(time.Duration) }).SetIntercharacterTimeout(20 * time.Millisecond)

	result := make(chan error, 1)
	go func() {
		_, receiveErr := codec.Receive()
		result <- receiveErr
	}()
	if _, err := server.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("expected network timeout, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Receive did not enforce T8")
	}
}

type shortWriter struct {
	bytes.Buffer
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) > 2 {
		p = p[:2]
	}
	return w.Buffer.Write(p)
}

func (w *shortWriter) Read([]byte) (int, error) { return 0, io.EOF }

func TestSecsIICodecHandlesShortWrites(t *testing.T) {
	rw := &shortWriter{}
	codec, err := SECSII().NewCodec(rw)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 3, 4, 5, 6, 7}
	if err := codec.Send(want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rw.Bytes(), want) {
		t.Fatalf("got %v, want %v", rw.Bytes(), want)
	}
}
