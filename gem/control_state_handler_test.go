package gem

import (
	"testing"
	"time"

	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func newTestGemHandler(t *testing.T, deviceType DeviceType, initial ControlState) *GemHandler {
	protocol := hsms.NewHsmsProtocol("127.0.0.1", 0, false, 0x100, "test")
	handler, err := NewGemHandler(Options{
		Protocol:            protocol,
		DeviceType:          deviceType,
		InitialControlState: initial,
	})
	if err != nil {
		t.Fatalf("NewGemHandler: %v", err)
	}
	return handler
}

func readAckValue(t *testing.T, msg *ast.DataMessage) int {
	if msg == nil {
		t.Fatal("nil message")
	}
	item, err := msg.Get(0)
	if err != nil {
		t.Fatalf("read ack: %v", err)
	}
	binary, ok := item.(*ast.BinaryNode)
	if !ok {
		t.Fatalf("unexpected ack item type %T", item)
	}
	values, ok := binary.Values().([]int)
	if !ok || len(values) == 0 {
		t.Fatalf("unexpected ack values %v", binary.Values())
	}
	return values[0]
}

func TestGemHandlerS1F15RemoteOffline(t *testing.T) {
	handler := newTestGemHandler(t, DeviceEquipment, ControlStateOnline)
	handler.control = NewControlStateMachine(ControlStateMachineOptions{
		InitialState:      ControlStateOnline,
		InitialOnlineMode: OnlineModeRemote,
	})

	if state := handler.ControlState(); state != ControlStateOnlineRemote {
		t.Fatalf("unexpected initial control state %s", state)
	}

	resp, err := handler.onS1F15(nil)
	if err != nil {
		t.Fatalf("onS1F15: %v", err)
	}
	if ack := readAckValue(t, resp); ack != 0 {
		t.Fatalf("unexpected OFLACK %d", ack)
	}
	if state := handler.ControlState(); state != ControlStateHostOffline {
		t.Fatalf("expected HOST_OFFLINE, got %s", state)
	}
}

func TestGemHandlerS1F17RequestOnline(t *testing.T) {
	handler := newTestGemHandler(t, DeviceEquipment, ControlStateHostOffline)
	handler.control = NewControlStateMachine(ControlStateMachineOptions{
		InitialState:      ControlStateHostOffline,
		InitialOnlineMode: OnlineModeRemote,
	})

	if state := handler.ControlState(); state != ControlStateHostOffline {
		t.Fatalf("unexpected initial control state %s", state)
	}

	resp, err := handler.onS1F17(nil)
	if err != nil {
		t.Fatalf("onS1F17: %v", err)
	}
	if ack := readAckValue(t, resp); ack != 0 {
		t.Fatalf("unexpected ONLACK %d", ack)
	}
	if state := handler.ControlState(); state != ControlStateOnlineRemote {
		t.Fatalf("expected ONLINE remote, got %s", state)
	}

	resp, err = handler.onS1F17(nil)
	if err != nil {
		t.Fatalf("onS1F17 second call: %v", err)
	}
	if ack := readAckValue(t, resp); ack != 2 {
		t.Fatalf("expected ONLACK 2, got %d", ack)
	}
	if state := handler.ControlState(); state != ControlStateOnlineRemote {
		t.Fatalf("state changed unexpectedly to %s", state)
	}
}

func TestGemHandlerAttemptOnlineFailsWithoutCommunication(t *testing.T) {
	handler := newTestGemHandler(t, DeviceEquipment, ControlStateEquipmentOffline)

	if err := handler.SwitchControlOnline(); err != nil {
		t.Fatalf("SwitchControlOnline: %v", err)
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if handler.ControlState() == ControlStateHostOffline {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected HOST_OFFLINE after attempt online, got %s", handler.ControlState())
}
