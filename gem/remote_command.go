package gem

import (
	"errors"
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

const cpackReservedMax = 63

type HCACKCode uint8

const (
	HCACKAcknowledge        HCACKCode = 0
	HCACKInvalidCommand     HCACKCode = 1
	HCACKCannotPerformNow   HCACKCode = 2
	HCACKParameterInvalid   HCACKCode = 3
	HCACKAcknowledgeLater   HCACKCode = 4
	HCACKAlreadyInCondition HCACKCode = 5
	HCACKNoObject           HCACKCode = 6
)

type CPACKCode uint8

const (
	CPACKParameterUnknown CPACKCode = 1
	CPACKValueIllegal     CPACKCode = 2
	CPACKFormatIllegal    CPACKCode = 3
)

type RemoteCommandParameterValue struct {
	Name  interface{}
	Value interface{}
}

type RemoteCommandParameter struct {
	Name       string
	Identifier interface{}
	Value      ast.ItemNode
}

type RemoteCommandParameterAck struct {
	Name interface{}
	Ack  CPACKCode
}

type RemoteCommandResult struct {
	HCACK         HCACKCode
	ParameterAcks []RemoteCommandParameterAck
}

type RemoteCommandRequest struct {
	Command    string
	CommandID  interface{}
	Parameters []RemoteCommandParameter
}

type RemoteCommandHandler func(RemoteCommandRequest) (RemoteCommandResult, error)

func (c HCACKCode) normalized() HCACKCode {
	if c > HCACKNoObject {
		return HCACKInvalidCommand
	}
	return c
}

func (c CPACKCode) normalized() CPACKCode {
	if c < CPACKParameterUnknown || c > CPACKCode(cpackReservedMax) {
		return CPACKParameterUnknown
	}
	return c
}

func (g *GemHandler) buildS2F41(command interface{}, params []RemoteCommandParameterValue) (*ast.DataMessage, error) {
	cmdInfo, err := newIDInfo(command)
	if err != nil {
		return nil, fmt.Errorf("gem: encode RCMD: %w", err)
	}

	encodedParams := make([]interface{}, 0, len(params))
	for _, param := range params {
		nameInfo, err := newIDInfo(param.Name)
		if err != nil {
			return nil, fmt.Errorf("gem: encode CPNAME: %w", err)
		}
		valueNode, err := itemNodeFromValue(param.Value)
		if err != nil {
			return nil, fmt.Errorf("gem: encode CPVAL for %v: %w", param.Name, err)
		}
		encodedParams = append(encodedParams, ast.NewListNode(nameInfo.node, valueNode))
	}

	direction := "H->E"
	if g.deviceType == DeviceEquipment {
		direction = "H<-E"
	}

	body := ast.NewListNode(cmdInfo.node, ast.NewListNode(encodedParams...))
	return ast.NewDataMessage("RemoteCommand", 2, 41, 1, direction, body), nil
}

func (g *GemHandler) buildS2F42(result RemoteCommandResult) (*ast.DataMessage, error) {
	paramNodes := make([]interface{}, 0, len(result.ParameterAcks))
	for _, ack := range result.ParameterAcks {
		nameInfo, err := newIDInfo(ack.Name)
		if err != nil {
			return nil, fmt.Errorf("gem: encode CPACK name: %w", err)
		}
		paramNodes = append(paramNodes, ast.NewListNode(nameInfo.node, ast.NewBinaryNode(int(ack.Ack.normalized()))))
	}

	direction := "H<-E"
	if g.deviceType == DeviceHost {
		direction = "H->E"
	}

	body := ast.NewListNode(ast.NewBinaryNode(int(result.HCACK.normalized())), ast.NewListNode(paramNodes...))
	return ast.NewDataMessage("RemoteCommandAck", 2, 42, 0, direction, body), nil
}

func parseRemoteCommand(msg *ast.DataMessage) (RemoteCommandRequest, error) {
	if msg == nil {
		return RemoteCommandRequest{}, errors.New("gem: nil S2F41 message")
	}

	cmdNode, err := msg.Get(0)
	if err != nil {
		return RemoteCommandRequest{}, fmt.Errorf("gem: missing RCMD: %w", err)
	}

	cmdInfo, err := newIDInfoFromNode(cmdNode)
	if err != nil {
		return RemoteCommandRequest{}, fmt.Errorf("gem: invalid RCMD: %w", err)
	}

	req := RemoteCommandRequest{
		Command:    fmt.Sprint(cmdInfo.raw),
		CommandID:  cmdInfo.raw,
		Parameters: make([]RemoteCommandParameter, 0),
	}

	paramsNode, err := msg.Get(1)
	if err != nil {
		return req, nil
	}

	list, ok := paramsNode.(*ast.ListNode)
	if !ok {
		return req, nil
	}

	for i := 0; i < list.Size(); i++ {
		entryNode, err := list.Get(i)
		if err != nil {
			continue
		}

		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() == 0 {
			continue
		}

		nameNode, err := entry.Get(0)
		if err != nil {
			continue
		}

		nameInfo, err := newIDInfoFromNode(nameNode)
		if err != nil {
			continue
		}

		value := ast.NewEmptyItemNode()
		if entry.Size() > 1 {
			if node, err := entry.Get(1); err == nil {
				value = node
			}
		}

		req.Parameters = append(req.Parameters, RemoteCommandParameter{
			Name:       fmt.Sprint(nameInfo.raw),
			Identifier: nameInfo.raw,
			Value:      value,
		})
	}

	return req, nil
}

func parseRemoteCommandAck(msg *ast.DataMessage) (RemoteCommandResult, error) {
	if msg == nil {
		return RemoteCommandResult{}, errors.New("gem: nil S2F42 message")
	}

	ackItem, err := msg.Get(0)
	if err != nil {
		return RemoteCommandResult{}, fmt.Errorf("gem: missing HCACK: %w", err)
	}

	hcackValue, err := readSingleBinaryValue(ackItem)
	if err != nil {
		return RemoteCommandResult{}, fmt.Errorf("gem: decode HCACK: %w", err)
	}

	result := RemoteCommandResult{HCACK: HCACKCode(hcackValue).normalized()}

	paramsNode, err := msg.Get(1)
	if err != nil {
		return result, nil
	}

	list, ok := paramsNode.(*ast.ListNode)
	if !ok {
		return result, nil
	}

	if list.Size() == 0 {
		return result, nil
	}

	result.ParameterAcks = make([]RemoteCommandParameterAck, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		entryNode, err := list.Get(i)
		if err != nil {
			continue
		}

		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() < 2 {
			continue
		}

		nameNode, err := entry.Get(0)
		if err != nil {
			continue
		}

		nameInfo, err := newIDInfoFromNode(nameNode)
		if err != nil {
			continue
		}

		ackNode, err := entry.Get(1)
		if err != nil {
			continue
		}

		ackValue, err := readSingleBinaryValue(ackNode)
		if err != nil {
			continue
		}

		result.ParameterAcks = append(result.ParameterAcks, RemoteCommandParameterAck{
			Name: nameInfo.raw,
			Ack:  CPACKCode(ackValue).normalized(),
		})
	}

	return result, nil
}

func itemNodeFromValue(value interface{}) (ast.ItemNode, error) {
	switch v := value.(type) {
	case nil:
		return ast.NewEmptyItemNode(), nil
	case ast.ItemNode:
		return v, nil
	case string:
		return ast.NewASCIINode(v), nil
	case []byte:
		ints := make([]interface{}, len(v))
		for i, b := range v {
			ints[i] = int(b)
		}
		return ast.NewBinaryNode(ints...), nil
	case bool:
		return ast.NewBooleanNode(v), nil
	case int:
		return ast.NewIntNode(signedByteSize(int64(v)), int64(v)), nil
	case int8:
		return ast.NewIntNode(signedByteSize(int64(v)), int64(v)), nil
	case int16:
		return ast.NewIntNode(signedByteSize(int64(v)), int64(v)), nil
	case int32:
		return ast.NewIntNode(signedByteSize(int64(v)), int64(v)), nil
	case int64:
		return ast.NewIntNode(signedByteSize(v), v), nil
	case uint:
		return ast.NewUintNode(unsignedByteSize(uint64(v)), uint64(v)), nil
	case uint8:
		return ast.NewUintNode(unsignedByteSize(uint64(v)), uint64(v)), nil
	case uint16:
		return ast.NewUintNode(unsignedByteSize(uint64(v)), uint64(v)), nil
	case uint32:
		return ast.NewUintNode(unsignedByteSize(uint64(v)), uint64(v)), nil
	case uint64:
		return ast.NewUintNode(unsignedByteSize(v), v), nil
	case float32:
		return ast.NewFloatNode(4, float64(v)), nil
	case float64:
		return ast.NewFloatNode(8, v), nil
	default:
		return nil, fmt.Errorf("gem: unsupported CPVAL type %T", value)
	}
}

func signedByteSize(value int64) int {
	switch {
	case value >= -(1<<7) && value < 1<<7:
		return 1
	case value >= -(1<<15) && value < 1<<15:
		return 2
	case value >= -(1<<31) && value < 1<<31:
		return 4
	default:
		return 8
	}
}

func unsignedByteSize(value uint64) int {
	switch {
	case value < 1<<8:
		return 1
	case value < 1<<16:
		return 2
	case value < 1<<32:
		return 4
	default:
		return 8
	}
}

func readSingleBinaryValue(item ast.ItemNode) (int, error) {
	switch node := item.(type) {
	case *ast.BinaryNode:
		values, ok := node.Values().([]int)
		if !ok || len(values) == 0 {
			return -1, errors.New("gem: binary node missing value")
		}
		return values[0], nil
	case *ast.ListNode:
		if node.Size() == 0 {
			return -1, errors.New("gem: empty list for binary value")
		}
		first, err := node.Get(0)
		if err != nil {
			return -1, err
		}
		return readSingleBinaryValue(first)
	default:
		return -1, fmt.Errorf("gem: expected binary node got %T", item)
	}
}
