package gem

import (
	"errors"
	"fmt"
	"log"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// RegisterStatusVariable installs an equipment-side status variable definition.
func (g *GemHandler) RegisterStatusVariable(variable *StatusVariable) error {
	if variable == nil {
		return errors.New("gem: status variable is nil")
	}
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}

	g.statusMu.Lock()
	defer g.statusMu.Unlock()

	key := variable.idKey()
	if _, exists := g.statusVars[key]; exists {
		return fmt.Errorf("gem: status variable %v already registered", variable.ID())
	}

	g.statusVars[key] = variable
	g.statusOrder = append(g.statusOrder, key)
	return nil
}

// RegisterEquipmentConstant installs an equipment-side equipment constant definition.
func (g *GemHandler) RegisterEquipmentConstant(constant *EquipmentConstant) error {
	if constant == nil {
		return errors.New("gem: equipment constant is nil")
	}
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}

	g.ecMu.Lock()
	defer g.ecMu.Unlock()

	key := constant.idKey()
	if _, exists := g.equipmentConstants[key]; exists {
		return fmt.Errorf("gem: equipment constant %v already registered", constant.ID())
	}

	g.equipmentConstants[key] = constant
	g.ecOrder = append(g.ecOrder, key)
	return nil
}

func (g *GemHandler) onS1F3(msg *ast.DataMessage) (*ast.DataMessage, error) {
	requests, err := parseIDRequestList(msg)
	if err != nil {
		g.logger.Println("failed to parse S1F3:", err)
		return g.buildS1F4(nil), nil
	}

	values := g.resolveStatusValues(requests)
	return g.buildS1F4(values), nil
}

func (g *GemHandler) onS1F11(msg *ast.DataMessage) (*ast.DataMessage, error) {
	requests, err := parseIDRequestList(msg)
	if err != nil {
		g.logger.Println("failed to parse S1F11:", err)
		return g.buildS1F12(nil), nil
	}

	entries := g.resolveStatusInfo(requests)
	return g.buildS1F12(entries), nil
}

func (g *GemHandler) onS2F13(msg *ast.DataMessage) (*ast.DataMessage, error) {
	requests, err := parseIDRequestList(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F13:", err)
		return g.buildS2F14(nil), nil
	}

	values := g.resolveEquipmentConstantValues(requests)
	return g.buildS2F14(values), nil
}

func (g *GemHandler) onS2F15(msg *ast.DataMessage) (*ast.DataMessage, error) {
	updates, err := parseEquipmentConstantUpdateList(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F15:", err)
		return g.buildS2F16(ECACKInvalidData), nil
	}

	ack := g.applyEquipmentConstantUpdates(updates)
	return g.buildS2F16(ack), nil
}

func (g *GemHandler) onS2F29(msg *ast.DataMessage) (*ast.DataMessage, error) {
	requests, err := parseIDRequestList(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F29:", err)
		return g.buildS2F30(nil), nil
	}

	entries := g.resolveEquipmentConstantInfo(requests)
	return g.buildS2F30(entries), nil
}

func (g *GemHandler) resolveStatusValues(requests []idRequest) []ast.ItemNode {
	g.statusMu.RLock()
	defer g.statusMu.RUnlock()

	if len(requests) == 0 {
		values := make([]ast.ItemNode, 0, len(g.statusOrder))
		for _, key := range g.statusOrder {
			if variable, ok := g.statusVars[key]; ok {
				values = append(values, safeStatusValue(variable, g.logger))
			}
		}
		return values
	}

	values := make([]ast.ItemNode, 0, len(requests))
	for _, req := range requests {
		if !req.ok {
			values = append(values, ast.NewEmptyItemNode())
			continue
		}
		if variable, ok := g.statusVars[req.info.key]; ok {
			values = append(values, safeStatusValue(variable, g.logger))
		} else {
			values = append(values, ast.NewEmptyItemNode())
		}
	}
	return values
}

func (g *GemHandler) resolveStatusInfo(requests []idRequest) []ast.ItemNode {
	g.statusMu.RLock()
	defer g.statusMu.RUnlock()

	if len(requests) == 0 {
		entries := make([]ast.ItemNode, 0, len(g.statusOrder))
		for _, key := range g.statusOrder {
			if variable, ok := g.statusVars[key]; ok {
				entries = append(entries, buildStatusInfoNode(variable))
			}
		}
		return entries
	}

	entries := make([]ast.ItemNode, 0, len(requests))
	for _, req := range requests {
		if !req.ok {
			entries = append(entries, ast.NewListNode(ast.NewEmptyItemNode(), ast.NewASCIINode(""), ast.NewASCIINode("")))
			continue
		}
		if variable, ok := g.statusVars[req.info.key]; ok {
			entries = append(entries, buildStatusInfoNode(variable))
		} else {
			idNode := req.info.node
			entries = append(entries, ast.NewListNode(idNode, ast.NewASCIINode(""), ast.NewASCIINode("")))
		}
	}
	return entries
}

func safeStatusValue(variable *StatusVariable, logger *log.Logger) ast.ItemNode {
	value, err := variable.Value()
	if err != nil || value == nil {
		if err != nil {
			logger.Printf("status variable %v value error: %v", variable.ID(), err)
		}
		return ast.NewEmptyItemNode()
	}
	return value
}

func buildStatusInfoNode(variable *StatusVariable) ast.ItemNode {
	return ast.NewListNode(
		variable.idNode(),
		ast.NewASCIINode(variable.Name),
		ast.NewASCIINode(variable.Unit),
	)
}

func (g *GemHandler) resolveEquipmentConstantValues(requests []idRequest) []ast.ItemNode {
	g.ecMu.RLock()
	defer g.ecMu.RUnlock()

	if len(requests) == 0 {
		values := make([]ast.ItemNode, 0, len(g.ecOrder))
		for _, key := range g.ecOrder {
			if constant, ok := g.equipmentConstants[key]; ok {
				values = append(values, safeEquipmentConstantValue(constant, g.logger))
			}
		}
		return values
	}

	values := make([]ast.ItemNode, 0, len(requests))
	for _, req := range requests {
		if !req.ok {
			values = append(values, ast.NewEmptyItemNode())
			continue
		}
		if constant, ok := g.equipmentConstants[req.info.key]; ok {
			values = append(values, safeEquipmentConstantValue(constant, g.logger))
		} else {
			values = append(values, ast.NewEmptyItemNode())
		}
	}
	return values
}

func (g *GemHandler) resolveEquipmentConstantInfo(requests []idRequest) []ast.ItemNode {
	g.ecMu.RLock()
	defer g.ecMu.RUnlock()

	if len(requests) == 0 {
		entries := make([]ast.ItemNode, 0, len(g.ecOrder))
		for _, key := range g.ecOrder {
			if constant, ok := g.equipmentConstants[key]; ok {
				entries = append(entries, buildEquipmentConstantInfoNode(constant))
			}
		}
		return entries
	}

	entries := make([]ast.ItemNode, 0, len(requests))
	for _, req := range requests {
		if !req.ok {
			entries = append(entries, emptyEquipmentConstantInfo(req))
			continue
		}
		if constant, ok := g.equipmentConstants[req.info.key]; ok {
			entries = append(entries, buildEquipmentConstantInfoNode(constant))
		} else {
			entries = append(entries, emptyEquipmentConstantInfo(req))
		}
	}
	return entries
}

func safeEquipmentConstantValue(constant *EquipmentConstant, logger *log.Logger) ast.ItemNode {
	value, err := constant.Value()
	if err != nil || value == nil {
		if err != nil {
			logger.Printf("equipment constant %v value error: %v", constant.ID(), err)
		}
		return ast.NewEmptyItemNode()
	}
	return value
}

func buildEquipmentConstantInfoNode(constant *EquipmentConstant) ast.ItemNode {
	minNode := constant.MinValue
	if minNode == nil {
		minNode = ast.NewEmptyItemNode()
	}
	maxNode := constant.MaxValue
	if maxNode == nil {
		maxNode = ast.NewEmptyItemNode()
	}
	defNode := constant.DefaultValue
	if defNode == nil {
		defNode = ast.NewEmptyItemNode()
	}

	return ast.NewListNode(
		constant.idNode(),
		ast.NewASCIINode(constant.Name),
		minNode,
		maxNode,
		defNode,
		ast.NewASCIINode(constant.Unit),
	)
}

func emptyEquipmentConstantInfo(req idRequest) ast.ItemNode {
	idNode := ast.NewEmptyItemNode()
	if req.ok {
		idNode = req.info.node
	}
	return ast.NewListNode(idNode, ast.NewASCIINode(""), ast.NewEmptyItemNode(), ast.NewEmptyItemNode(), ast.NewEmptyItemNode(), ast.NewASCIINode(""))
}

func parseIDRequestList(msg *ast.DataMessage) ([]idRequest, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	item, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := item.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload, got %T", item)
	}

	if list.Size() == 0 {
		return []idRequest{}, nil
	}

	result := make([]idRequest, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		node, err := list.Get(i)
		if err != nil {
			return nil, err
		}
		info, parseErr := newIDInfoFromNode(node)
		if parseErr != nil {
			result = append(result, idRequest{ok: false})
			continue
		}
		result = append(result, idRequest{info: info, ok: true})
	}
	return result, nil
}

type equipmentConstantUpdate struct {
	id    idInfo
	value ast.ItemNode
	ok    bool
}

func parseEquipmentConstantUpdateList(msg *ast.DataMessage) ([]equipmentConstantUpdate, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	item, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := item.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload, got %T", item)
	}

	updates := make([]equipmentConstantUpdate, 0, list.Size())
	for i := 0; i < list.Size(); i++ {
		entryNode, err := list.Get(i)
		if err != nil {
			return nil, err
		}

		entryList, ok := entryNode.(*ast.ListNode)
		if !ok || entryList.Size() < 2 {
			updates = append(updates, equipmentConstantUpdate{ok: false})
			continue
		}

		idNode, err := entryList.Get(0)
		if err != nil {
			return nil, err
		}

		valueNode, err := entryList.Get(1)
		if err != nil {
			return nil, err
		}

		info, parseErr := newIDInfoFromNode(idNode)
		if parseErr != nil {
			updates = append(updates, equipmentConstantUpdate{ok: false})
			continue
		}

		updates = append(updates, equipmentConstantUpdate{ok: true, id: info, value: valueNode})
	}
	return updates, nil
}

func (g *GemHandler) applyEquipmentConstantUpdates(updates []equipmentConstantUpdate) ECACKCode {
	if len(updates) == 0 {
		return ECACKAccepted
	}

	g.ecMu.RLock()
	defer g.ecMu.RUnlock()

	for _, upd := range updates {
		if !upd.ok {
			return ECACKInvalidData
		}
		if _, ok := g.equipmentConstants[upd.id.key]; !ok {
			return ECACKDoesNotExist
		}
	}

	for _, upd := range updates {
		constant := g.equipmentConstants[upd.id.key]
		if err := constant.ApplyValue(upd.value); err != nil {
			g.logger.Printf("equipment constant %v update rejected: %v", constant.ID(), err)
			return ECACKValidationError
		}
	}

	return ECACKAccepted
}
