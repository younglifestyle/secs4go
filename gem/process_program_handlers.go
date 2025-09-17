package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// RegisterProcessProgram stores a process program locally on the equipment side.
func (g *GemHandler) RegisterProcessProgram(ppid interface{}, body string) error {
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}
	program, err := newProcessProgram(ppid, body)
	if err != nil {
		return err
	}
	g.processStore.put(program)
	return nil
}

// ListProcessPrograms returns a snapshot of stored process programs.
func (g *GemHandler) ListProcessPrograms() []ProcessProgram {
	items := g.processStore.list()
	result := make([]ProcessProgram, 0, len(items))
	for _, pp := range items {
		result = append(result, *pp)
	}
	return result
}

// SetProcessProgramUploadHandler registers a callback invoked when the host uploads a process program (S7F3).
func (g *GemHandler) SetProcessProgramUploadHandler(handler ProcessProgramUploadHandler) {
	g.processUploadHandler = handler
}

// SetProcessProgramRequestHandler registers a callback used to serve S7F5 requests.
func (g *GemHandler) SetProcessProgramRequestHandler(handler ProcessProgramRequestHandler) {
	g.processRequestHandler = handler
}

func (g *GemHandler) onS7F3(msg *ast.DataMessage) (*ast.DataMessage, error) {
	if msg == nil {
		return g.buildS7F4(1), nil
	}
	ppidNode, err := msg.Get(0)
	if err != nil {
		return g.buildS7F4(1), nil
	}
	ppidInfo, err := newIDInfoFromNode(ppidNode)
	if err != nil {
		return g.buildS7F4(1), nil
	}
	bodyNode, err := msg.Get(1)
	if err != nil {
		return g.buildS7F4(1), nil
	}
	body := ""
	if ascii, ok := bodyNode.(*ast.ASCIINode); ok {
		if val, ok := ascii.Values().(string); ok {
			body = val
		}
	}

	ack := 0
	if g.processUploadHandler != nil {
		ack = g.processUploadHandler(ppidInfo.raw, body)
	}
	if ack == 0 {
		program, err := newProcessProgram(ppidInfo.raw, body)
		if err != nil {
			ack = 1
		} else {
			g.processStore.put(program)
		}
	}

	return g.buildS7F4(ack), nil
}

func (g *GemHandler) onS7F5(msg *ast.DataMessage) (*ast.DataMessage, error) {
	if msg == nil {
		return g.buildS7F6(ast.NewEmptyItemNode(), "", 1), nil
	}
	ppidNode, err := msg.Get(0)
	if err != nil {
		return g.buildS7F6(ast.NewEmptyItemNode(), "", 1), nil
	}
	ppidInfo, err := newIDInfoFromNode(ppidNode)
	if err != nil {
		return g.buildS7F6(ast.NewEmptyItemNode(), "", 1), nil
	}

	body := ""
	ack := 0
	if g.processRequestHandler != nil {
		body, ack = g.processRequestHandler(ppidInfo.raw)
	} else {
		if program, ok := g.processStore.get(ppidInfo.key); ok {
			body = program.Body
		} else {
			ack = 1
		}
	}

	return g.buildS7F6(ppidInfo.node, body, ack), nil
}

func (g *GemHandler) processProgram(ppid interface{}) (*ProcessProgram, error) {
	key, err := ensureProcessProgramKey(ppid)
	if err != nil {
		return nil, err
	}
	if program, ok := g.processStore.get(key); ok {
		return program, nil
	}
	return nil, fmt.Errorf("process program %v not found", ppid)
}
