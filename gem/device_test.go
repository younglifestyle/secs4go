package gem

import (
	"github.com/younglifestyle/secs4go/hsms"
	"testing"
	"time"
)

func Test_Data(t *testing.T) {
	t.Skip("manual interoperability test; requires an equipment peer on port 5022")
	opts := Options{
		Protocol:   hsms.NewHsmsProtocol("127.0.0.1", 5022, true, 0, "sample"),
		DeviceType: DeviceHost,
	}
	handler, err := NewGemHandler(opts)
	if err != nil {
		t.Fatal(err)
	}
	handler.Enable()
	defer handler.Disable()
	if handler.WaitForCommunicating(30 * time.Second) {
		t.Log("GEM session established")
	}
}
