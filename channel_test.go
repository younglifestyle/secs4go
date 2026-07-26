package link

import (
	"testing"
	"time"
)

func TestChannelFetchCallbackCanRemove(t *testing.T) {
	channel := NewChannel()
	session := NewSession(&noopCodec{}, 0)
	channel.Put(session.ID(), session)
	done := make(chan struct{})
	go func() {
		channel.Fetch(func(item *Session) { channel.Remove(item.ID()) })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Fetch deadlocked when callback modified the channel")
	}
}

type noopCodec struct{}

func (*noopCodec) Receive() (interface{}, error) { return nil, nil }
func (*noopCodec) Send(interface{}) error        { return nil }
func (*noopCodec) Close() error                  { return nil }
