package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// -----------------------------------------------------------------------------
// HCACK codes for S2F42 (Host Command Acknowledge)
// 0  Acknowledge
// 1  Denied, invalid command
// 2  Denied, cannot perform now
// 3  Denied, parameter invalid
// 4  Acknowledge, will finish later
// 5  Rejected, already in condition
// 6  No such object
// 7–63 Reserved
// -----------------------------------------------------------------------------
const (
	HCACKAcknowledge            = 0
	HCACKDeniedInvalidCommand   = 1
	HCACKDeniedCannotPerformNow = 2
	HCACKDeniedParamInvalid     = 3
	HCACKAckWillFinishLater     = 4
	HCACKRejectedAlreadyOk      = 5
	HCACKNoSuchObject           = 6
	// 7–63 are reserved
	hcackMaxReserved = 63
)

// RemoteCommandRequest is passed to equipment-side handlers when a host issues S2F41.
type RemoteCommandRequest struct {
	Command    string
	Parameters []string
}

// RemoteCommandHandler processes a remote command and returns HCACK (0-5).
type RemoteCommandHandler func(RemoteCommandRequest) (int, error)

func (g *GemHandler) buildS2F41(command string, params []string) *ast.DataMessage {
	paramNodes := make([]interface{}, 0, len(params))
	for _, p := range params {
		paramNodes = append(paramNodes, ast.NewASCIINode(p))
	}

	body := ast.NewListNode(
		ast.NewASCIINode(command),
		ast.NewListNode(paramNodes...),
	)

	return ast.NewDataMessage("RemoteCommand", 2, 41, 1, "H->E", body)
}

func (g *GemHandler) buildS2F42(hcack int) *ast.DataMessage {
	if hcack < 0 {
		hcack = 1
	}
	body := ast.NewListNode(ast.NewBinaryNode(hcack))
	direction := "H<-E"
	if g.deviceType == DeviceHost {
		direction = "H->E"
	}
	return ast.NewDataMessage("RemoteCommandAck", 2, 42, 0, direction, body)
}

func parseRemoteCommand(msg *ast.DataMessage) (RemoteCommandRequest, error) {
	var req RemoteCommandRequest

	cmdNode, err := msg.Get(0)
	if err != nil {
		return req, fmt.Errorf("gem: malformed S2F41 RCMD: %w", err)
	}
	ascii, ok := cmdNode.(*ast.ASCIINode)
	if !ok {
		return req, fmt.Errorf("gem: S2F41 RCMD not ASCII")
	}
	if text, ok := ascii.Values().(string); ok {
		req.Command = text
	}

	paramsNode, err := msg.Get(1)
	if err != nil {
		return req, nil
	}
	list, ok := paramsNode.(*ast.ListNode)
	if !ok {
		return req, nil
	}

	params := make([]string, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		item, err := list.Get(i)
		if err != nil {
			continue
		}
		if asciiParam, ok := item.(*ast.ASCIINode); ok {
			if text, ok := asciiParam.Values().(string); ok {
				params = append(params, text)
			}
		}
	}

	req.Parameters = params
	return req, nil
}

func readHCACK(msg *ast.DataMessage) (int, error) {
	item, err := msg.Get()
	if err != nil {
		return -1, err
	}

	binary, ok := item.(*ast.BinaryNode)
	if !ok {
		if list, ok := item.(*ast.ListNode); ok {
			if list.Size() == 0 {
				return -1, fmt.Errorf("HCACK missing value")
			}
			first, err := list.Get(0)
			if err != nil {
				return -1, err
			}
			binNode, ok := first.(*ast.BinaryNode)
			if !ok {
				return -1, fmt.Errorf("HCACK not binary (got %T)", first)
			}
			binary = binNode
		} else {
			return -1, fmt.Errorf("HCACK not binary (got %T)", item)
		}
	}

	values, ok := binary.Values().([]int)
	if !ok || len(values) == 0 {
		return -1, fmt.Errorf("HCACK missing value")
	}
	return values[0], nil
}
