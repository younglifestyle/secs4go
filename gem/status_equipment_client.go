package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// ReportDefinitionRequest describes the payload for S2F33 Define Report.
type ReportDefinitionRequest struct {
	ReportID interface{}
	VIDs     []interface{}
}

// EventReportLinkRequest describes the payload for S2F35 Link Event Report.
type EventReportLinkRequest struct {
	CEID      interface{}
	ReportIDs []interface{}
}

// RequestStatusVariables queries the equipment for the current value of the supplied SVIDs.
// ids must be non-empty and contain non-negative integers or ASCII strings.
func (g *GemHandler) RequestStatusVariables(ids ...interface{}) ([]StatusValue, error) {
	if g.deviceType != DeviceHost {
		return nil, ErrOperationNotSupported
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("gem: at least one SVID required")
	}
	if err := g.ensureCommunicating(); err != nil {
		return nil, err
	}

	idInfos := make([]idInfo, 0, len(ids))
	for _, id := range ids {
		info, err := newIDInfo(id)
		if err != nil {
			return nil, err
		}
		idInfos = append(idInfos, info)
	}

	request := g.buildS1F3(idInfos)
	response, err := g.protocol.SendAndWait(request)
	if err != nil {
		return nil, err
	}

	values, err := parseStatusValueResponse(response, idInfos)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// RequestStatusVariableInfo retrieves status variable name/unit metadata via S1F11/S1F12.
// When ids is empty the equipment returns the complete namelist.
func (g *GemHandler) RequestStatusVariableInfo(ids ...interface{}) ([]StatusVariableInfo, error) {
	if g.deviceType != DeviceHost {
		return nil, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return nil, err
	}

	idInfos := make([]idInfo, 0, len(ids))
	for _, id := range ids {
		info, err := newIDInfo(id)
		if err != nil {
			return nil, err
		}
		idInfos = append(idInfos, info)
	}

	request := g.buildS1F11(idInfos)
	response, err := g.protocol.SendAndWait(request)
	if err != nil {
		return nil, err
	}

	infos, err := parseStatusInfoResponse(response)
	if err != nil {
		return nil, err
	}
	return infos, nil
}

// RequestEquipmentConstants fetches equipment constant values via S2F13/S2F14.
func (g *GemHandler) RequestEquipmentConstants(ids ...interface{}) ([]EquipmentConstantValue, error) {
	if g.deviceType != DeviceHost {
		return nil, ErrOperationNotSupported
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("gem: at least one ECID required")
	}
	if err := g.ensureCommunicating(); err != nil {
		return nil, err
	}

	idInfos := make([]idInfo, 0, len(ids))
	for _, id := range ids {
		info, err := newIDInfo(id)
		if err != nil {
			return nil, err
		}
		idInfos = append(idInfos, info)
	}

	request := g.buildS2F13(idInfos)
	response, err := g.protocol.SendAndWait(request)
	if err != nil {
		return nil, err
	}

	values, err := parseEquipmentConstantValueResponse(response, idInfos)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// RequestEquipmentConstantInfo retrieves EC metadata via S2F29/S2F30.
// When ids is empty the full namelist is returned.
func (g *GemHandler) RequestEquipmentConstantInfo(ids ...interface{}) ([]EquipmentConstantInfo, error) {
	if g.deviceType != DeviceHost {
		return nil, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return nil, err
	}

	idInfos := make([]idInfo, 0, len(ids))
	for _, id := range ids {
		info, err := newIDInfo(id)
		if err != nil {
			return nil, err
		}
		idInfos = append(idInfos, info)
	}

	request := g.buildS2F29(idInfos)
	response, err := g.protocol.SendAndWait(request)
	if err != nil {
		return nil, err
	}

	infos, err := parseEquipmentConstantInfoResponse(response)
	if err != nil {
		return nil, err
	}
	return infos, nil
}

// DefineReports installs or clears report definitions on the equipment using S2F33.
func (g *GemHandler) DefineReports(defs ...ReportDefinitionRequest) (int, error) {
	if g.deviceType != DeviceHost {
		return -1, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return -1, err
	}

	msg, err := g.buildS2F33(defs)
	if err != nil {
		return -1, err
	}

	resp, err := g.protocol.SendAndWait(msg)
	if err != nil {
		return -1, err
	}
	ack, err := readBinaryAck(resp)
	if err != nil {
		return -1, err
	}
	return ack, nil
}

// LinkEventReports associates collection events with existing reports using S2F35.
func (g *GemHandler) LinkEventReports(links ...EventReportLinkRequest) (int, error) {
	if g.deviceType != DeviceHost {
		return -1, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return -1, err
	}

	msg, err := g.buildS2F35(links)
	if err != nil {
		return -1, err
	}

	resp, err := g.protocol.SendAndWait(msg)
	if err != nil {
		return -1, err
	}
	ack, err := readBinaryAck(resp)
	if err != nil {
		return -1, err
	}
	return ack, nil
}

// EnableEventReports toggles event reporting through S2F37.
func (g *GemHandler) EnableEventReports(enable bool, ceids ...interface{}) (int, error) {
	if g.deviceType != DeviceHost {
		return -1, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return -1, err
	}

	infoSlice := make([]idInfo, 0, len(ceids))
	for _, id := range ceids {
		info, err := newIDInfo(id)
		if err != nil {
			return -1, err
		}
		infoSlice = append(infoSlice, info)
	}

	resp, err := g.protocol.SendAndWait(g.buildS2F37(enable, infoSlice))
	if err != nil {
		return -1, err
	}
	ack, err := readBinaryAck(resp)
	if err != nil {
		return -1, err
	}
	return ack, nil
}

// RequestCollectionEventReport requests current data for a CEID via S6F15/S6F16.
func (g *GemHandler) RequestCollectionEventReport(ceid interface{}) (EventReport, error) {
	var report EventReport
	if g.deviceType != DeviceHost {
		return report, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return report, err
	}

	info, err := newIDInfo(ceid)
	if err != nil {
		return report, err
	}

	resp, err := g.protocol.SendAndWait(g.buildS6F15(info))
	if err != nil {
		return report, err
	}

	return parseEventReportMessage(resp)
}

// UploadProcessProgram sends an S7F3 to store a process program on the equipment.
func (g *GemHandler) UploadProcessProgram(ppid interface{}, body string) (int, error) {
	if g.deviceType != DeviceHost {
		return -1, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return -1, err
	}

	info, err := newIDInfo(ppid)
	if err != nil {
		return -1, err
	}

	resp, err := g.protocol.SendAndWait(g.buildS7F3(info, body))
	if err != nil {
		return -1, err
	}
	ack, err := readBinaryAck(resp)
	if err != nil {
		return -1, err
	}
	return ack, nil
}

// RequestProcessProgram retrieves a process program via S7F5/S7F6.
func (g *GemHandler) RequestProcessProgram(ppid interface{}) (string, int, error) {
	if g.deviceType != DeviceHost {
		return "", -1, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return "", -1, err
	}

	info, err := newIDInfo(ppid)
	if err != nil {
		return "", -1, err
	}

	resp, err := g.protocol.SendAndWait(g.buildS7F5(info))
	if err != nil {
		return "", -1, err
	}

	return parseProcessProgramResponse(resp)
}

// SendEquipmentConstantValues issues an S2F15 update and returns the received acknowledgement code.
func (g *GemHandler) SendEquipmentConstantValues(updates []EquipmentConstantUpdate) (int, error) {
	if g.deviceType != DeviceHost {
		return -1, ErrOperationNotSupported
	}
	if len(updates) == 0 {
		return 0, nil
	}
	if err := g.ensureCommunicating(); err != nil {
		return -1, err
	}

	msg, err := g.buildS2F15(updates)
	if err != nil {
		return -1, err
	}

	response, err := g.protocol.SendAndWait(msg)
	if err != nil {
		return -1, err
	}

	ack, err := parseEquipmentConstantAck(response)
	if err != nil {
		return -1, err
	}
	return ack, nil
}

func parseStatusValueResponse(msg *ast.DataMessage, ids []idInfo) ([]StatusValue, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil response")
	}

	item, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := item.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload, got %T", item)
	}

	if list.Size() != len(ids) {
		return nil, fmt.Errorf("unexpected value count %d (expected %d)", list.Size(), len(ids))
	}

	values := make([]StatusValue, 0, len(ids))
	for i, id := range ids {
		node, err := list.Get(i)
		if err != nil {
			return nil, err
		}
		values = append(values, StatusValue{ID: id.raw, Value: node})
	}
	return values, nil
}

func parseStatusInfoResponse(msg *ast.DataMessage) ([]StatusVariableInfo, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil response")
	}

	item, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := item.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload, got %T", item)
	}

	infos := make([]StatusVariableInfo, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		entryNode, err := list.Get(i)
		if err != nil {
			return nil, err
		}

		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() < 3 {
			return nil, fmt.Errorf("malformed S1F12 entry")
		}

		idNode, err := entry.Get(0)
		if err != nil {
			return nil, err
		}
		info, err := newIDInfoFromNode(idNode)
		if err != nil {
			return nil, err
		}

		nameNode, err := entry.Get(1)
		if err != nil {
			return nil, err
		}
		unitNode, err := entry.Get(2)
		if err != nil {
			return nil, err
		}

		name := ""
		if nameNode.Type() == "ascii" {
			if v, ok := nameNode.Values().(string); ok {
				name = v
			}
		}

		unit := ""
		if unitNode.Type() == "ascii" {
			if v, ok := unitNode.Values().(string); ok {
				unit = v
			}
		}

		infos = append(infos, StatusVariableInfo{ID: info.raw, Name: name, Unit: unit})
	}

	return infos, nil
}

func parseEquipmentConstantValueResponse(msg *ast.DataMessage, ids []idInfo) ([]EquipmentConstantValue, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil response")
	}

	item, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := item.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload, got %T", item)
	}

	if list.Size() != len(ids) {
		return nil, fmt.Errorf("unexpected value count %d (expected %d)", list.Size(), len(ids))
	}

	values := make([]EquipmentConstantValue, 0, len(ids))
	for i, id := range ids {
		node, err := list.Get(i)
		if err != nil {
			return nil, err
		}
		values = append(values, EquipmentConstantValue{ID: id.raw, Value: node})
	}
	return values, nil
}

func parseEquipmentConstantInfoResponse(msg *ast.DataMessage) ([]EquipmentConstantInfo, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil response")
	}

	item, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := item.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload, got %T", item)
	}

	infos := make([]EquipmentConstantInfo, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		entryNode, err := list.Get(i)
		if err != nil {
			return nil, err
		}

		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() < 6 {
			return nil, fmt.Errorf("malformed S2F30 entry")
		}

		idNode, err := entry.Get(0)
		if err != nil {
			return nil, err
		}
		info, err := newIDInfoFromNode(idNode)
		if err != nil {
			return nil, err
		}

		name := ""
		if nameNode, err := entry.Get(1); err == nil && nameNode.Type() == "ascii" {
			if v, ok := nameNode.Values().(string); ok {
				name = v
			}
		}

		unit := ""
		if unitNode, err := entry.Get(5); err == nil && unitNode.Type() == "ascii" {
			if v, ok := unitNode.Values().(string); ok {
				unit = v
			}
		}

		minNode, _ := entry.Get(2)
		maxNode, _ := entry.Get(3)
		defNode, _ := entry.Get(4)

		infos = append(infos, EquipmentConstantInfo{
			ID:      info.raw,
			Name:    name,
			Unit:    unit,
			Min:     minNode,
			Max:     maxNode,
			Default: defNode,
		})
	}

	return infos, nil
}

func parseEquipmentConstantAck(msg *ast.DataMessage) (int, error) {
	return readBinaryAck(msg)
}

func readBinaryAck(msg *ast.DataMessage) (int, error) {
	if msg == nil {
		return -1, fmt.Errorf("nil response")
	}

	node, err := msg.Get(0)
	if err != nil {
		return -1, err
	}

	binary, ok := node.(*ast.BinaryNode)
	if !ok {
		return -1, fmt.Errorf("expected binary ack, got %T", node)
	}

	values, ok := binary.Values().([]int)
	if !ok || len(values) == 0 {
		return -1, fmt.Errorf("invalid ack payload")
	}

	return values[0], nil
}

func parseProcessProgramResponse(msg *ast.DataMessage) (string, int, error) {
	if msg == nil {
		return "", -1, fmt.Errorf("nil response")
	}

	list, err := msg.Get()
	if err != nil {
		return "", -1, err
	}

	entries, ok := list.(*ast.ListNode)
	if !ok || entries.Size() < 3 {
		return "", -1, fmt.Errorf("malformed S7F6 payload")
	}

	bodyNode, err := entries.Get(1)
	if err != nil {
		return "", -1, err
	}
	body := ""
	if ascii, ok := bodyNode.(*ast.ASCIINode); ok {
		if val, ok := ascii.Values().(string); ok {
			body = val
		}
	}

	ackNode, err := entries.Get(2)
	if err != nil {
		return "", -1, err
	}
	ackBinary, ok := ackNode.(*ast.BinaryNode)
	if !ok {
		return "", -1, fmt.Errorf("expected binary ack, got %T", ackNode)
	}
	values, ok := ackBinary.Values().([]int)
	if !ok || len(values) == 0 {
		return "", -1, fmt.Errorf("invalid ack payload")
	}

	return body, values[0], nil
}
