package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// Alarm Enhancement handlers (S5F3/F4, S5F5/F6, S5F7/F8) - Equipment side

// onS5F3 handles Enable/Disable Alarm Request (Equipment side).
// Host → Equipment: S5F3 W
//   <L[2]
//     <ALED (BINARY[1] - 128=Enable, 0=Disable)>
//     <L[n] <ALID (4-byte)> ... >
//   >
// Equipment → Host: S5F4 ACKC5 (BINARY[1] - 0=Accepted, 1-63=Error)
func (g *GemHandler) onS5F3(msg *ast.DataMessage) (*ast.DataMessage, error) {
	// Parse ALED (enable/disable flag)
	aledNode, err := msg.Get(0)
	if err != nil {
		return g.buildS5F4(1), nil // Error: malformed
	}

	aledValue, err := readSingleBinaryValue(aledNode)
	if err != nil {
		return g.buildS5F4(1), nil
	}

	enable := aledValue == 128

	// Parse ALID list
	alidListNode, err := msg.Get(1)
	if err != nil {
		return g.buildS5F4(1), nil
	}

	listNode, ok := alidListNode.(*ast.ListNode)
	if !ok {
		return g.buildS5F4(1), nil
	}

	// Extract alarm IDs
	alarmIDs := make([]int, 0)
	for i := 0; i < listNode.Size(); i++ {
		alidNode, err := listNode.Get(i)
		if err != nil {
			continue
		}

		var id int
		switch typed := alidNode.(type) {
		case *ast.UintNode:
			values, _ := typed.Values().([]uint64)
			if len(values) > 0 {
				id = int(values[0])
			}
		case *ast.IntNode:
			values, _ := typed.Values().([]int64)
			if len(values) > 0 {
				id = int(values[0])
			}
		default:
			return g.buildS5F4(1), nil
		}

		alarmIDs = append(alarmIDs, id)
	}

	// Validate all alarm IDs exist
	g.alarmMu.Lock()
	for _, id := range alarmIDs {
		if _, exists := g.alarms[id]; !exists {
			g.alarmMu.Unlock()
			return g.buildS5F4(1), nil // At least one ALID does not exist
		}
	}

	// Update alarm enabled state
	for _, id := range alarmIDs {
		alarm := g.alarms[id]
		alarm.Enabled = enable
		g.alarms[id] = alarm
	}
	g.alarmMu.Unlock()

	return g.buildS5F4(0), nil // Accepted
}

// onS5F5 handles List Alarms Request (Equipment side).
// Host → Equipment: S5F5 (empty)
// Equipment → Host: S5F6 W
//   <L[n]
//     <L[3]
//       <ALCD (BINARY[1] - alarm code byte)>
//       <ALID (4-byte)>
//       <ALTX (ASCII - alarm text)>
//     >
//     ...
//   >
func (g *GemHandler) onS5F5(msg *ast.DataMessage) (*ast.DataMessage, error) {
	alarms := g.getAlarmList()
	return g.buildS5F6(alarms), nil
}

// onS5F7 handles List Enabled Alarms Request (Equipment side).
// Host → Equipment: S5F7 (empty)
// Equipment → Host: S5F8 W (same format as S5F6, but only enabled alarms)
func (g *GemHandler) onS5F7(msg *ast.DataMessage) (*ast.DataMessage, error) {
	enabledAlarms := g.getEnabledAlarmList()
	return g.buildS5F8(enabledAlarms), nil
}

// buildS5F4 creates S5F4 response with ACKC5 code.
// ACKC5 codes:
//   0 - Accepted
//   1 - At least one ALID does not exist
//   2 - Busy, try later
//   3 - At least one ALID cannot be enabled
func (g *GemHandler) buildS5F4(ackc5 byte) *ast.DataMessage {
	body := ast.NewBinaryNode(int(ackc5))
	return ast.NewDataMessage("EnableAlarmAck", 5, 4, 0, "E->H", body)
}

// buildS5F6 creates S5F6 response with alarm list.
// ALCD format:
//   Bit 7: 0=Not set, 1=Set
//   Bit 6: 0=Disabled, 1=Enabled
//   Bits 0-5: Reserved
func (g *GemHandler) buildS5F6(alarms []AlarmInfo) *ast.DataMessage {
	items := make([]interface{}, len(alarms))
	for i, a := range alarms {
		alcd := 0
		if a.Set {
			alcd |= 0x80 // Bit 7: Set
		}
		if a.Enabled {
			alcd |= 0x40 // Bit 6: Enabled
		}

		items[i] = ast.NewListNode(
			ast.NewBinaryNode(alcd),
			ast.NewUintNode(4, a.ID),
			ast.NewASCIINode(a.Text),
		)
	}

	body := ast.NewListNode(items...)
	return ast.NewDataMessage("AlarmListData", 5, 6, 0, "E->H", body)
}

// buildS5F8 creates S5F8 response with enabled alarm list (same format as S5F6).
func (g *GemHandler) buildS5F8(alarms []AlarmInfo) *ast.DataMessage {
	items := make([]interface{}, len(alarms))
	for i, a := range alarms {
		alcd := 0
		if a.Set {
			alcd |= 0x80
		}
		if a.Enabled {
			alcd |= 0x40
		}

		items[i] = ast.NewListNode(
			ast.NewBinaryNode(alcd),
			ast.NewUintNode(4, a.ID),
			ast.NewASCIINode(a.Text),
		)
	}

	body := ast.NewListNode(items...)
	return ast.NewDataMessage("EnabledAlarmListData", 5, 8, 0, "E->H", body)
}

// Public APIs for alarm management (Equipment side)

// EnableAlarm enables an alarm for reporting (equipment side).
func (g *GemHandler) EnableAlarm(alarmID int) error {
	g.alarmMu.Lock()
	defer g.alarmMu.Unlock()

	alarm, exists := g.alarms[alarmID]
	if !exists {
		return fmt.Errorf("gem: unknown alarm %d", alarmID)
	}

	alarm.Enabled = true
	g.alarms[alarmID] = alarm
	return nil
}

// DisableAlarm disables an alarm from reporting (equipment side).
func (g *GemHandler) DisableAlarm(alarmID int) error {
	g.alarmMu.Lock()
	defer g.alarmMu.Unlock()

	alarm, exists := g.alarms[alarmID]
	if !exists {
		return fmt.Errorf("gem: unknown alarm %d", alarmID)
	}

	alarm.Enabled = false
	g.alarms[alarmID] = alarm
	return nil
}
