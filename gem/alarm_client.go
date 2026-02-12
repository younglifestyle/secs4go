package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// Alarm management APIs (Host side)

// SendEnableAlarm sends S5F3 to enable or disable alarms on the equipment.
// ACKC5 codes: 0=Accepted, 1=ALID not exist, 2=Busy, 3=Cannot enable
func (h *GemHandler) SendEnableAlarm(alarmIDs []int, enable bool) (byte, error) {
	if h.deviceType != DeviceHost {
		return 0, ErrOperationNotSupported
	}
	if err := h.ensureCommunicating(); err != nil {
		return 0, err
	}

	// Build ALID list
	alidItems := make([]interface{}, len(alarmIDs))
	for i, id := range alarmIDs {
		alidItems[i] = ast.NewUintNode(4, id)
	}

	// ALED: 128=Enable, 0=Disable
	aled := 0
	if enable {
		aled = 128
	}

	body := ast.NewListNode(
		ast.NewBinaryNode(aled),
		ast.NewListNode(alidItems...),
	)

	req := ast.NewDataMessage("EnableAlarmReq", 5, 3, 1, "H->E", body)
	resp, err := h.protocol.SendAndWait(req)
	if err != nil {
		return 0, fmt.Errorf("gem: S5F3 failed: %w", err)
	}

	// Parse ACKC5 from S5F4
	ackc5, err := readBinaryAck(resp)
	if err != nil {
		return 0, fmt.Errorf("gem: failed to parse S5F4: %w", err)
	}

	return byte(ackc5), nil
}

// RequestAlarmList sends S5F5 to request all registered alarms from equipment.
func (h *GemHandler) RequestAlarmList() ([]AlarmInfo, error) {
	if h.deviceType != DeviceHost {
		return nil, ErrOperationNotSupported
	}
	if err := h.ensureCommunicating(); err != nil {
		return nil, err
	}

	req := ast.NewDataMessage("AlarmListReq", 5, 5, 1, "H->E", ast.NewListNode())
	resp, err := h.protocol.SendAndWait(req)
	if err != nil {
		return nil, fmt.Errorf("gem: S5F5 failed: %w", err)
	}

	// Parse S5F6 response
	return parseAlarmListResponse(resp)
}

// RequestEnabledAlarmList sends S5F7 to request only enabled alarms from equipment.
func (h *GemHandler) RequestEnabledAlarmList() ([]AlarmInfo, error) {
	if h.deviceType != DeviceHost {
		return nil, ErrOperationNotSupported
	}
	if err := h.ensureCommunicating(); err != nil {
		return nil, err
	}

	req := ast.NewDataMessage("EnabledAlarmListReq", 5, 7, 1, "H->E", ast.NewListNode())
	resp, err := h.protocol.SendAndWait(req)
	if err != nil {
		return nil, fmt.Errorf("gem: S5F7 failed: %w", err)
	}

	// Parse S5F8 response (same format as S5F6)
	return parseAlarmListResponse(resp)
}

// parseAlarmListResponse parses S5F6 or S5F8 alarm list response.
// Format: <L[n] <L[3] <ALCD> <ALID> <ALTX> > ... >
func parseAlarmListResponse(msg *ast.DataMessage) ([]AlarmInfo, error) {
	listNode, err := msg.Get()
	if err != nil {
		return nil, fmt.Errorf("gem: malformed alarm list: %w", err)
	}

	outerList, ok := listNode.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("gem: alarm list not a list")
	}

	alarms := make([]AlarmInfo, 0, outerList.Size())
	for i := 0; i < outerList.Size(); i++ {
		alarmNode, err := outerList.Get(i)
		if err != nil {
			continue
		}

		alarmList, ok := alarmNode.(*ast.ListNode)
		if !ok || alarmList.Size() < 3 {
			continue
		}

		// Parse ALCD (byte 0)
		alcdNode, err := alarmList.Get(0)
		if err != nil {
			continue
		}
		alcd, err := readSingleBinaryValue(alcdNode)
		if err != nil {
			continue
		}

		// Parse ALID (index 1)
		alidNode, err := alarmList.Get(1)
		if err != nil {
			continue
		}
		var alid int
		switch typed := alidNode.(type) {
		case *ast.UintNode:
			values, _ := typed.Values().([]uint64)
			if len(values) > 0 {
				alid = int(values[0])
			}
		case *ast.IntNode:
			values, _ := typed.Values().([]int64)
			if len(values) > 0 {
				alid = int(values[0])
			}
		default:
			continue
		}

		// Parse ALTX (index 2)
		altxNode, err := alarmList.Get(2)
		if err != nil {
			continue
		}
		var text string
		if asciiNode, ok := altxNode.(*ast.ASCIINode); ok {
			if str, ok := asciiNode.Values().(string); ok {
				text = str
			}
		}

		alarms = append(alarms, AlarmInfo{
			ID:      alid,
			Text:    text,
			Set:     (alcd & 0x80) != 0, // Bit 7: Set
			Enabled: (alcd & 0x40) != 0, // Bit 6: Enabled
		})
	}

	return alarms, nil
}
