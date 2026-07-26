package gem

import (
	"testing"

	"github.com/younglifestyle/secs4go/hsms"
)

func TestDeviceIDOverridesProtocolSessionBeforeEnable(t *testing.T) {
	protocol := hsms.NewHsmsProtocol("127.0.0.1", 5000, true, 1, "test")
	handler, err := NewGemHandler(Options{
		Protocol:   protocol,
		DeviceType: DeviceHost,
		DeviceID:   7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if protocol.SessionID() != 7 || handler.deviceID != 7 {
		t.Fatalf("device ID was not applied: protocol=%d handler=%d", protocol.SessionID(), handler.deviceID)
	}
}

func TestZeroDeviceIDKeepsProtocolSession(t *testing.T) {
	protocol := hsms.NewHsmsProtocol("127.0.0.1", 5000, true, 9, "test")
	handler, err := NewGemHandler(Options{Protocol: protocol, DeviceType: DeviceHost})
	if err != nil {
		t.Fatal(err)
	}
	if handler.deviceID != 9 {
		t.Fatalf("handler device ID = %d, want 9", handler.deviceID)
	}
}
