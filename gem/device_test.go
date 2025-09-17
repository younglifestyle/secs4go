package gem

import (
	"github.com/younglifestyle/secs4go/hsms"
	"log"
	"testing"
	"time"
)

func Test_Data(t *testing.T) {
	opts := Options{
		Protocol:   hsms.NewHsmsProtocol("127.0.0.1", 5022, true, 0xFFFF, "sample"),
		DeviceType: DeviceHost,
	}
	handler, err := NewGemHandler(opts)
	if err != nil {
		log.Fatal(err)
	}
	handler.Enable()
	if handler.WaitForCommunicating(30 * time.Second) {
		log.Println("GEM session established")
	}
}
