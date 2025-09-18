package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func (g *GemHandler) buildS1F3(ids []idInfo) *ast.DataMessage {
	items := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		items = append(items, id.node)
	}
	body := ast.NewListNode(items...)
	return ast.NewDataMessage("SelectedEquipmentStatusRequest", 1, 3, 1, "H->E", body)
}

func (g *GemHandler) buildS1F4(values []ast.ItemNode) *ast.DataMessage {
	nodes := make([]interface{}, 0, len(values))
	for _, value := range values {
		if value == nil {
			nodes = append(nodes, ast.NewEmptyItemNode())
		} else {
			nodes = append(nodes, value)
		}
	}
	body := ast.NewListNode(nodes...)
	return ast.NewDataMessage("SelectedEquipmentStatusData", 1, 4, 0, "H<-E", body)
}

func (g *GemHandler) buildS1F11(ids []idInfo) *ast.DataMessage {
	items := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		items = append(items, id.node)
	}
	body := ast.NewListNode(items...)
	return ast.NewDataMessage("StatusVariableNamelistRequest", 1, 11, 1, "H->E", body)
}

func (g *GemHandler) buildS1F12(entries []ast.ItemNode) *ast.DataMessage {
	nodes := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes, entry)
	}
	body := ast.NewListNode(nodes...)
	return ast.NewDataMessage("StatusVariableNamelist", 1, 12, 0, "H<-E", body)
}

func (g *GemHandler) buildS2F13(ids []idInfo) *ast.DataMessage {
	items := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		items = append(items, id.node)
	}
	body := ast.NewListNode(items...)
	return ast.NewDataMessage("EquipmentConstantRequest", 2, 13, 1, "H->E", body)
}

func (g *GemHandler) buildS2F14(values []ast.ItemNode) *ast.DataMessage {
	nodes := make([]interface{}, 0, len(values))
	for _, value := range values {
		if value == nil {
			nodes = append(nodes, ast.NewEmptyItemNode())
		} else {
			nodes = append(nodes, value)
		}
	}
	body := ast.NewListNode(nodes...)
	return ast.NewDataMessage("EquipmentConstantData", 2, 14, 0, "H<-E", body)
}

func (g *GemHandler) buildS2F15(updates []EquipmentConstantUpdate) (*ast.DataMessage, error) {
	entries := make([]interface{}, 0, len(updates))
	for _, upd := range updates {
		info, err := newIDInfo(upd.ID)
		if err != nil {
			return nil, err
		}
		if upd.Value == nil {
			return nil, fmt.Errorf("equipment constant %v value is nil", upd.ID)
		}
		entries = append(entries, ast.NewListNode(info.node, upd.Value))
	}
	body := ast.NewListNode(entries...)
	return ast.NewDataMessage("EquipmentConstantSend", 2, 15, 1, "H->E", body), nil
}

func (g *GemHandler) buildS2F16(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("EquipmentConstantAcknowledge", 2, 16, 0, "H<-E", body)
}

func (g *GemHandler) buildS2F29(ids []idInfo) *ast.DataMessage {
	items := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		items = append(items, id.node)
	}
	body := ast.NewListNode(items...)
	return ast.NewDataMessage("EquipmentConstantNamelistRequest", 2, 29, 1, "H->E", body)
}

func (g *GemHandler) buildS2F30(entries []ast.ItemNode) *ast.DataMessage {
	nodes := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		nodes = append(nodes, entry)
	}
	body := ast.NewListNode(nodes...)
	return ast.NewDataMessage("EquipmentConstantNamelist", 2, 30, 0, "H<-E", body)
}

func (g *GemHandler) buildS2F33(defs []ReportDefinitionRequest) (*ast.DataMessage, error) {
	reports := make([]interface{}, 0, len(defs))
	for _, def := range defs {
		rptInfo, err := newIDInfo(def.ReportID)
		if err != nil {
			return nil, err
		}
		vidInfos, err := ensureIDInfoSlice(def.VIDs)
		if err != nil {
			return nil, err
		}
		vidNodes := make([]interface{}, 0, len(vidInfos))
		for _, vid := range vidInfos {
			vidNodes = append(vidNodes, vid.node)
		}
		reports = append(reports, ast.NewListNode(rptInfo.node, ast.NewListNode(vidNodes...)))
	}
	reportList := ast.NewListNode(reports...)
	body := ast.NewListNode(ast.NewUintNode(1, 0), reportList)
	return ast.NewDataMessage("DefineReport", 2, 33, 1, "H->E", body), nil
}

func (g *GemHandler) buildS2F34(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("DefineReportAcknowledge", 2, 34, 0, "H<-E", body)
}

func (g *GemHandler) buildS2F35(links []EventReportLinkRequest) (*ast.DataMessage, error) {
	items := make([]interface{}, 0, len(links))
	for _, link := range links {
		ceInfo, err := newIDInfo(link.CEID)
		if err != nil {
			return nil, err
		}
		rptInfos, err := ensureIDInfoSlice(link.ReportIDs)
		if err != nil {
			return nil, err
		}
		rptNodes := make([]interface{}, 0, len(rptInfos))
		for _, rpt := range rptInfos {
			rptNodes = append(rptNodes, rpt.node)
		}
		items = append(items, ast.NewListNode(ceInfo.node, ast.NewListNode(rptNodes...)))
	}
	linkList := ast.NewListNode(items...)
	body := ast.NewListNode(ast.NewUintNode(1, 0), linkList)
	return ast.NewDataMessage("LinkEventReport", 2, 35, 1, "H->E", body), nil
}

func (g *GemHandler) buildS2F36(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("LinkEventReportAcknowledge", 2, 36, 0, "H<-E", body)
}

func (g *GemHandler) buildS2F37(enable bool, ceids []idInfo) *ast.DataMessage {
	ceNodes := make([]interface{}, 0, len(ceids))
	for _, ce := range ceids {
		ceNodes = append(ceNodes, ce.node)
	}
	body := ast.NewListNode(ast.NewBooleanNode(enable), ast.NewListNode(ceNodes...))
	return ast.NewDataMessage("EnableEventReport", 2, 37, 1, "H->E", body)
}

func (g *GemHandler) buildS2F38(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("EnableEventReportAcknowledge", 2, 38, 0, "H<-E", body)
}

func (g *GemHandler) buildS6F11(dataID int, ceNode ast.ItemNode, reports []ast.ItemNode) *ast.DataMessage {
	reportNodes := make([]interface{}, 0, len(reports))
	for _, rpt := range reports {
		reportNodes = append(reportNodes, rpt)
	}
	body := ast.NewListNode(ast.NewBinaryNode(dataID), ceNode, ast.NewListNode(reportNodes...))
	return ast.NewDataMessage("EventReport", 6, 11, 0, "H->E", body)
}

func (g *GemHandler) buildS6F12(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("EventReportAcknowledge", 6, 12, 0, "H<-E", body)
}

func (g *GemHandler) buildS6F15(ceid idInfo) *ast.DataMessage {
	return ast.NewDataMessage("EventReportRequest", 6, 15, 1, "H->E", ceid.node)
}

func (g *GemHandler) buildS6F16(dataID int, ceNode ast.ItemNode, reports []ast.ItemNode) *ast.DataMessage {
	reportNodes := make([]interface{}, 0, len(reports))
	for _, rpt := range reports {
		reportNodes = append(reportNodes, rpt)
	}
	var ceItem ast.ItemNode = ast.NewEmptyItemNode()
	if ceNode != nil {
		ceItem = ceNode
	}
	byteSize := byteSizeForUint(uint64(dataID))
	body := ast.NewListNode(ast.NewUintNode(byteSize, dataID), ceItem, ast.NewListNode(reportNodes...))
	return ast.NewDataMessage("EventReportData", 6, 16, 0, "H<-E", body)
}

func (g *GemHandler) buildS7F3(ppid idInfo, programBody string) *ast.DataMessage {
	payload := ast.NewListNode(ppid.node, ast.NewASCIINode(programBody))
	return ast.NewDataMessage("ProcessProgramSend", 7, 3, 1, "H->E", payload)
}

func (g *GemHandler) buildS7F4(ack int) *ast.DataMessage {
	body := ast.NewListNode(ast.NewBinaryNode(ack))
	return ast.NewDataMessage("ProcessProgramAcknowledge", 7, 4, 0, "H<-E", body)
}

func (g *GemHandler) buildS7F5(ppid idInfo) *ast.DataMessage {
	return ast.NewDataMessage("ProcessProgramRequest", 7, 5, 1, "H->E", ppid.node)
}

func (g *GemHandler) buildS7F6(ppid ast.ItemNode, programBody string, ack int) *ast.DataMessage {
	ppidNode := ppid
	if ppidNode == nil {
		ppidNode = ast.NewEmptyItemNode()
	}
	bodyNode := ast.NewASCIINode(programBody)
	if ack != 0 {
		bodyNode = ast.NewASCIINode("")
	}
	payload := ast.NewListNode(ppidNode, bodyNode, ast.NewBinaryNode(ack))
	return ast.NewDataMessage("ProcessProgramData", 7, 6, 0, "H<-E", payload)
}
