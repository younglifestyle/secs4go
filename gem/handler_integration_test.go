package gem_test

import (
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	"github.com/younglifestyle/secs4go/gem"
	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
	hsmsparser "github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/parser/hsms"
)

func TestParseS1F14Sample(t *testing.T) {
	body := ast.NewListNode(
		ast.NewBinaryNode(0),
		ast.NewListNode(),
	)
	msg := ast.NewHSMSDataMessage("", 1, 14, 0, "H<-E", body, 0x0100, []byte{0, 0, 0, 1})
	raw := msg.ToBytes()
	if _, ok := hsmsparser.Parse(raw); !ok {
		t.Fatalf("parse failed for %s", hex.EncodeToString(raw))
	}
}

func TestGemHandlerIntegration(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	port := 6000 + rand.Intn(500)

	t.Logf("starting GEM integration test on port %d", port)

	equipmentProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, false, 0x0100, "equipment")
	hostProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, true, 0x0100, "host")

	equipmentProtocol.Timeouts().SetLinktest(60)
	hostProtocol.Timeouts().SetLinktest(60)

	equipmentHandler, err := gem.NewGemHandler(gem.Options{
		Protocol:   equipmentProtocol,
		DeviceType: gem.DeviceEquipment,
		DeviceID:   0,
		MDLN:       "EQUIP",
		SOFTREV:    "1.0.0",
	})
	if err != nil {
		t.Fatalf("create equipment handler: %v", err)
	}

	hostHandler, err := gem.NewGemHandler(gem.Options{
		Protocol:   hostProtocol,
		DeviceType: gem.DeviceHost,
		DeviceID:   0,
		MDLN:       "HOST",
		SOFTREV:    "1.0.0",
	})
	if err != nil {
		t.Fatalf("create host handler: %v", err)
	}

	defer equipmentHandler.Disable()
	defer hostHandler.Disable()

	alarmCh := make(chan gem.AlarmEvent, 1)
	hostHandler.Events().AlarmReceived.AddCallback(func(data map[string]interface{}) {
		if evt, ok := data["alarm"].(gem.AlarmEvent); ok {
			select {
			case alarmCh <- evt:
			default:
			}
		}
	})

	remoteCh := make(chan gem.RemoteCommandRequest, 1)
	equipmentHandler.SetRemoteCommandHandler(func(req gem.RemoteCommandRequest) (int, error) {
		select {
		case remoteCh <- req:
		default:
		}
		return 0, nil
	})

	equipmentHandler.RegisterAlarm(gem.Alarm{ID: 1001, Text: "Door Open"})

	equipmentHandler.Enable()
	time.Sleep(200 * time.Millisecond)
	hostHandler.Enable()

	if !hostHandler.WaitForCommunicating(10 * time.Second) {
		t.Fatal("host failed to reach communicating state")
	}
	if !equipmentHandler.WaitForCommunicating(10 * time.Second) {
		t.Fatal("equipment failed to reach communicating state")
	}

	t.Log("host and equipment reached communicating state")

	if err := equipmentHandler.RaiseAlarm(1001, true); err != nil {
		t.Fatalf("raise alarm: %v", err)
	}

	select {
	case evt := <-alarmCh:
		if evt.ID != 1001 || !evt.Set {
			t.Fatalf("unexpected alarm event: %+v", evt)
		}
		t.Logf("alarm event propagated successfully: ID=%d set=%t", evt.ID, evt.Set)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for alarm event")
	}

	ack, err := hostHandler.SendRemoteCommand("RESET", []string{"IMMEDIATE"})
	if err != nil {
		t.Fatalf("send remote command: %v", err)
	}
	if ack != 0 {
		t.Fatalf("unexpected HCACK %d", ack)
	}
	t.Logf("remote command acknowledged with HCACK=%d", ack)

	select {
	case req := <-remoteCh:
		if req.Command != "RESET" {
			t.Fatalf("unexpected remote command: %+v", req)
		}
		t.Logf("equipment remote command handler invoked: %s %v", req.Command, req.Parameters)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for remote command handler")
	}

	t.Log("GEM integration flow completed successfully")
}
