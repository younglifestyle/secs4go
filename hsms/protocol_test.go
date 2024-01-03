package hsms

import (
	"fmt"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	timer := time.AfterFunc(2*time.Second, func() {
		fmt.Println("test")
	})
	defer timer.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(time.Hour)
		done <- true
	}()

	for {
		select {
		case <-timer.C:
			reset := timer.Reset(2 * time.Second)
			fmt.Println("test : ", reset)
		case <-done:
			return
		}
	}
}

func TestRun(t *testing.T) {
	protocol := NewHsmsProtocol("127.0.0.1", 5555, true, 1, "test")
	protocol.enable()

	time.Sleep(time.Hour)
}

func TestServerRun(t *testing.T) {
	protocol := NewHsmsProtocol("127.0.0.1", 5555, false, 1, "test")
	protocol.enable()

	time.Sleep(time.Hour)
}
