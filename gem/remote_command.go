package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
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
	node, err := msg.Get(0)
	if err != nil {
		return -1, err
	}
	binary, ok := node.(*ast.BinaryNode)
	if !ok {
		return -1, fmt.Errorf("HCACK not binary")
	}
	values, ok := binary.Values().([]int)
	if !ok || len(values) == 0 {
		return -1, fmt.Errorf("HCACK missing value")
	}
	return values[0], nil
}
