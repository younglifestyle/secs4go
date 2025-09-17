package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// Alarm represents a GEM alarm definition for equipment side.
type Alarm struct {
	ID   int
	Text string
}

// AlarmEvent describes an alarm notification received from the remote peer.
type AlarmEvent struct {
	ID   int
	Text string
	Set  bool
}

func (g *GemHandler) RegisterAlarm(alarm Alarm) {
	g.alarmMu.Lock()
	defer g.alarmMu.Unlock()

	if g.alarms == nil {
		g.alarms = make(map[int]Alarm)
	}
	g.alarms[alarm.ID] = alarm
}

// RaiseAlarm notifies the remote peer about an alarm state change (equipment only).
func (g *GemHandler) RaiseAlarm(alarmID int, set bool) error {
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return err
	}

	alarm, ok := g.lookupAlarm(alarmID)
	if !ok {
		return fmt.Errorf("gem: unknown alarm %d", alarmID)
	}

	msg := g.buildS5F1(alarm, set)
	return g.protocol.SendDataMessage(msg)
}

// ClearAlarm clears a previously raised alarm (equipment only).
func (g *GemHandler) ClearAlarm(alarmID int) error {
	return g.RaiseAlarm(alarmID, false)
}

func (g *GemHandler) lookupAlarm(id int) (Alarm, bool) {
	g.alarmMu.RLock()
	defer g.alarmMu.RUnlock()

	alarm, ok := g.alarms[id]
	return alarm, ok
}

func (g *GemHandler) buildS5F1(alarm Alarm, set bool) *ast.DataMessage {
	alcd := 0
	if set {
		alcd = 1
	}

	body := ast.NewListNode(
		ast.NewBinaryNode(alcd),
		ast.NewUintNode(2, alarm.ID),
		ast.NewASCIINode(alarm.Text),
	)

	direction := "H<-E"
	if g.deviceType == DeviceHost {
		direction = "H->E"
	}

	return ast.NewDataMessage("AlarmReport", 5, 1, 0, direction, body)
}

func (g *GemHandler) buildS5F2(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("AlarmAck", 5, 2, 0, "H->E", body)
}

func parseAlarmMessage(msg *ast.DataMessage) (AlarmEvent, error) {
	var event AlarmEvent

	alcdNode, err := msg.Get(0)
	if err != nil {
		return event, fmt.Errorf("gem: malformed S5F1 ALCD: %w", err)
	}
	binary, ok := alcdNode.(*ast.BinaryNode)
	if !ok {
		return event, fmt.Errorf("gem: S5F1 ALCD not binary")
	}
	alcdValues, ok := binary.Values().([]int)
	if !ok || len(alcdValues) == 0 {
		return event, fmt.Errorf("gem: S5F1 ALCD missing value")
	}
	event.Set = alcdValues[0] != 0

	alidNode, err := msg.Get(1)
	if err != nil {
		return event, fmt.Errorf("gem: malformed S5F1 ALID: %w", err)
	}
	switch typed := alidNode.(type) {
	case *ast.UintNode:
		values, _ := typed.Values().([]uint64)
		if len(values) > 0 {
			event.ID = int(values[0])
		}
	case *ast.IntNode:
		values, _ := typed.Values().([]int64)
		if len(values) > 0 {
			event.ID = int(values[0])
		}
	default:
		return event, fmt.Errorf("gem: unsupported ALID type %T", alidNode)
	}

	altxNode, err := msg.Get(2)
	if err == nil {
		if ascii, ok := altxNode.(*ast.ASCIINode); ok {
			if text, ok := ascii.Values().(string); ok {
				event.Text = text
			}
		}
	}

	return event, nil
}
