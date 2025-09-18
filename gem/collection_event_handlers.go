package gem

import (
	"errors"
	"fmt"
	"log"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

const (
	drAckOK             = 0
	drAckRPTIDRedefined = 1
	drAckVIDUnknown     = 2
)

const (
	lrAckOK           = 0
	lrAckCEIDUnknown  = 1
	lrAckRPTIDUnknown = 2
	lrAckCEIDLinked   = 3
)

const (
	erAckAccepted    = 0
	erAckCEIDUnknown = 1
)

// RegisterDataVariable installs an equipment-side data variable definition.
func (g *GemHandler) RegisterDataVariable(variable *DataVariable) error {
	if variable == nil {
		return errors.New("gem: data variable is nil")
	}
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}

	g.dataVarMu.Lock()
	defer g.dataVarMu.Unlock()

	key := variable.idKey()
	if _, exists := g.dataVars[key]; exists {
		return fmt.Errorf("gem: data variable %v already registered", variable.ID())
	}

	g.dataVars[key] = variable
	g.dataVarOrder = append(g.dataVarOrder, key)
	return nil
}

// RegisterCollectionEvent installs an equipment-side collection event definition.
func (g *GemHandler) RegisterCollectionEvent(event *CollectionEvent) error {
	if event == nil {
		return errors.New("gem: collection event is nil")
	}
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}

	g.collectionMu.Lock()
	defer g.collectionMu.Unlock()

	key := event.idKey()
	if _, exists := g.collectionEvents[key]; exists {
		return fmt.Errorf("gem: collection event %v already registered", event.ID())
	}

	g.collectionEvents[key] = event
	g.collectionOrder = append(g.collectionOrder, key)
	return nil
}

// TriggerCollectionEvent emits an S6F11 for each supplied CEID that is linked and enabled.
func (g *GemHandler) TriggerCollectionEvent(ids ...interface{}) error {
	if g.deviceType != DeviceEquipment {
		return ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return err
	}

	if len(ids) == 0 {
		return fmt.Errorf("gem: at least one CEID required")
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		info, err := newIDInfo(id)
		if err != nil {
			return err
		}
		keys = append(keys, info.key)
	}

	for _, key := range keys {
		go g.sendCollectionEvent(key)
	}
	return nil
}

func (g *GemHandler) sendCollectionEvent(key string) {
	reports, dataID, ceNode, err := g.buildCollectionEventPayload(key)
	if err != nil {
		g.logger.Printf("collection event build failed: %v", err)
		return
	}
	if reports == nil {
		return
	}

	msg := g.buildS6F11(dataID, ceNode, reports)
	if sendErr := g.protocol.SendDataMessage(msg); sendErr != nil {
		g.logger.Printf("failed to send S6F11: %v", sendErr)
	}
}

func (g *GemHandler) buildCollectionEventPayload(key string) ([]ast.ItemNode, int, ast.ItemNode, error) {
	g.collectionMu.RLock()
	event, ok := g.collectionEvents[key]
	g.collectionMu.RUnlock()
	if !ok {
		return nil, 0, nil, fmt.Errorf("unknown collection event %s", key)
	}

	g.reportMu.RLock()
	link, linked := g.eventLinks[key]
	g.reportMu.RUnlock()
	if !linked || !link.enabled || len(link.reports) == 0 {
		return nil, 0, nil, nil
	}

	g.reportMu.RLock()
	reportKeys := append([]string{}, link.reports...)
	g.reportMu.RUnlock()

	reportNodes := make([]ast.ItemNode, 0, len(reportKeys))
	for _, rptKey := range reportKeys {
		g.reportMu.RLock()
		report, exists := g.reports[rptKey]
		g.reportMu.RUnlock()
		if !exists {
			continue
		}

		values := g.collectReportValues(report)
		reportNodes = append(reportNodes, ast.NewListNode(report.idNode(), ast.NewListNode(values...)))
	}

	if len(reportNodes) == 0 {
		return nil, 0, nil, nil
	}

	return reportNodes, 1, event.idNode(), nil
}

func (g *GemHandler) collectReportValues(report *ReportDefinition) []interface{} {
	values := make([]interface{}, 0, len(report.vidKeys))
	for _, vidKey := range report.vidKeys {
		if node := g.resolveVIDValue(vidKey); node != nil {
			values = append(values, node)
		} else {
			values = append(values, ast.NewEmptyItemNode())
		}
	}
	return values
}

func (g *GemHandler) resolveVIDValue(key string) ast.ItemNode {
	g.statusMu.RLock()
	if variable, ok := g.statusVars[key]; ok {
		g.statusMu.RUnlock()
		return safeStatusValue(variable, g.logger)
	}
	g.statusMu.RUnlock()

	g.dataVarMu.RLock()
	variable, ok := g.dataVars[key]
	g.dataVarMu.RUnlock()
	if ok {
		return safeDataVariableValue(variable, g.logger)
	}

	return nil
}

func safeDataVariableValue(variable *DataVariable, logger *log.Logger) ast.ItemNode {
	value, err := variable.Value()
	if err != nil || value == nil {
		if err != nil {
			logger.Printf("data variable %v value error: %v", variable.ID(), err)
		}
		return ast.NewEmptyItemNode()
	}
	return value
}

func (g *GemHandler) onS2F33(msg *ast.DataMessage) (*ast.DataMessage, error) {
	reports, err := parseReportDefinitionList(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F33:", err)
		return g.buildS2F34(drAckVIDUnknown), nil
	}

	ack := g.handleReportDefinitions(reports)
	return g.buildS2F34(ack), nil
}

func (g *GemHandler) handleReportDefinitions(reports []reportDefinitionMessage) int {
	if len(reports) == 0 {
		g.reportMu.Lock()
		g.eventLinks = make(map[string]*collectionEventLink)
		g.reports = make(map[string]*ReportDefinition)
		g.reportMu.Unlock()
		return drAckOK
	}

	g.reportMu.Lock()
	defer g.reportMu.Unlock()

	for _, rpt := range reports {
		if _, exists := g.reports[rpt.id.key]; exists && len(rpt.vids) > 0 {
			return drAckRPTIDRedefined
		}
	}

	for _, rpt := range reports {
		for _, vid := range rpt.vids {
			if !g.vidExists(vid.key) {
				return drAckVIDUnknown
			}
		}
	}

	for _, rpt := range reports {
		if len(rpt.vids) == 0 {
			g.removeReportUnlocked(rpt.id.key)
			continue
		}
		definition, err := newReportDefinition(rpt.id.raw, rpt.vids)
		if err != nil {
			return drAckVIDUnknown
		}
		g.reports[rpt.id.key] = definition
	}

	return drAckOK
}

func (g *GemHandler) removeReportUnlocked(reportKey string) {
	delete(g.reports, reportKey)
	for ceKey, link := range g.eventLinks {
		link.removeReport(reportKey)
		if len(link.reports) == 0 {
			delete(g.eventLinks, ceKey)
		}
	}
}

func (g *GemHandler) vidExists(key string) bool {
	g.statusMu.RLock()
	if _, ok := g.statusVars[key]; ok {
		g.statusMu.RUnlock()
		return true
	}
	g.statusMu.RUnlock()

	g.dataVarMu.RLock()
	_, ok := g.dataVars[key]
	g.dataVarMu.RUnlock()
	return ok
}

func (g *GemHandler) onS2F35(msg *ast.DataMessage) (*ast.DataMessage, error) {
	links, err := parseEventReportLinkList(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F35:", err)
		return g.buildS2F36(lrAckCEIDUnknown), nil
	}

	ack := g.handleEventReportLinks(links)
	return g.buildS2F36(ack), nil
}

func (g *GemHandler) handleEventReportLinks(links []eventReportLinkMessage) int {
	g.collectionMu.RLock()
	defer g.collectionMu.RUnlock()

	g.reportMu.Lock()
	defer g.reportMu.Unlock()

	for _, link := range links {
		if _, ok := g.collectionEvents[link.ceid.key]; !ok {
			return lrAckCEIDUnknown
		}
		for _, rpt := range link.rptids {
			if _, ok := g.reports[rpt.key]; !ok {
				return lrAckRPTIDUnknown
			}
			if existing, ok := g.eventLinks[link.ceid.key]; ok {
				for _, existingRpt := range existing.reports {
					if existingRpt == rpt.key {
						return lrAckCEIDLinked
					}
				}
			}
		}
	}

	for _, link := range links {
		if len(link.rptids) == 0 {
			delete(g.eventLinks, link.ceid.key)
			continue
		}
		reportKeys := make([]string, 0, len(link.rptids))
		for _, rpt := range link.rptids {
			reportKeys = append(reportKeys, rpt.key)
		}
		if existing, ok := g.eventLinks[link.ceid.key]; ok {
			existing.reports = append(existing.reports, reportKeys...)
		} else {
			g.eventLinks[link.ceid.key] = newCollectionEventLink(reportKeys)
		}
	}

	return lrAckOK
}

func (g *GemHandler) onS2F37(msg *ast.DataMessage) (*ast.DataMessage, error) {
	command, err := parseEventEnableMessage(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F37:", err)
		return g.buildS2F38(erAckCEIDUnknown), nil
	}

	ak := g.setCollectionEventState(command.enable, command.ceids)
	if !ak {
		return g.buildS2F38(erAckCEIDUnknown), nil
	}
	return g.buildS2F38(erAckAccepted), nil
}

func (g *GemHandler) setCollectionEventState(enable bool, ceids []idInfo) bool {
	g.reportMu.Lock()
	defer g.reportMu.Unlock()

	if len(ceids) == 0 {
		for _, link := range g.eventLinks {
			link.enabled = enable
		}
		return true
	}

	for _, ce := range ceids {
		if link, ok := g.eventLinks[ce.key]; ok {
			link.enabled = enable
		} else {
			return false
		}
	}
	return true
}

func (g *GemHandler) onS6F15(msg *ast.DataMessage) (*ast.DataMessage, error) {
	req, err := parseEventReportRequest(msg)
	if err != nil {
		g.logger.Println("failed to parse S6F15:", err)
		return g.buildS6F16(1, nil, nil), nil
	}

	reports, dataID, ceNode, buildErr := g.buildCollectionEventPayload(req.key)
	if buildErr != nil {
		g.logger.Println("failed to build S6F16 payload:", buildErr)
		return g.buildS6F16(1, req.node, nil), nil
	}

	if reports == nil {
		return g.buildS6F16(0, req.node, nil), nil
	}

	return g.buildS6F16(dataID, ceNode, reports), nil
}

func (g *GemHandler) onS6F11(msg *ast.DataMessage) (*ast.DataMessage, error) {
	report, err := parseEventReportMessage(msg)
	if err != nil {
		g.logger.Println("failed to parse S6F11:", err)
		return g.buildS6F12(1), nil
	}

	if g.events.EventReportReceived != nil {
		g.events.EventReportReceived.Fire(map[string]interface{}{"report": report})
	}
	return g.buildS6F12(0), nil
}

// Data structures used during parsing of S2F33/S2F35 messages.
type reportDefinitionMessage struct {
	id   idInfo
	vids []idInfo
}

type eventReportLinkMessage struct {
	ceid   idInfo
	rptids []idInfo
}

type eventEnableMessage struct {
	enable bool
	ceids  []idInfo
}

func parseReportDefinitionList(msg *ast.DataMessage) ([]reportDefinitionMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	root, err := msg.Get()
	if err != nil {
		return nil, err
	}

	list, ok := root.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload for S2F33")
	}

	if list.Size() == 0 {
		return nil, nil
	}

	var entries *ast.ListNode
	firstNode, err := list.Get(0)
	if err != nil {
		return nil, err
	}
	if _, ok := firstNode.(*ast.ListNode); ok {
		entries = list
	} else {
		if list.Size() < 2 {
			return nil, fmt.Errorf("malformed S2F33 payload")
		}
		secondNode, err := list.Get(1)
		if err != nil {
			return nil, err
		}
		entryList, ok := secondNode.(*ast.ListNode)
		if !ok {
			return nil, fmt.Errorf("malformed S2F33 report list")
		}
		entries = entryList
	}

	result := make([]reportDefinitionMessage, 0, entries.Size())
	for i := 0; i < entries.Size(); i++ {
		entryNode, err := entries.Get(i)
		if err != nil {
			return nil, err
		}
		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() < 2 {
			return nil, fmt.Errorf("malformed S2F33 entry")
		}
		rptIDNode, err := entry.Get(0)
		if err != nil {
			return nil, err
		}
		rptID, err := newIDInfoFromNode(rptIDNode)
		if err != nil {
			return nil, err
		}
		vidsNode, err := entry.Get(1)
		if err != nil {
			return nil, err
		}
		vidList, ok := vidsNode.(*ast.ListNode)
		if !ok {
			return nil, fmt.Errorf("malformed S2F33 VID list")
		}
		vidInfos := make([]idInfo, 0, vidList.Size())
		for idx := 0; idx < vidList.Size(); idx++ {
			vidNode, err := vidList.Get(idx)
			if err != nil {
				return nil, err
			}
			vidInfo, err := newIDInfoFromNode(vidNode)
			if err != nil {
				return nil, err
			}
			vidInfos = append(vidInfos, vidInfo)
		}

		result = append(result, reportDefinitionMessage{id: rptID, vids: vidInfos})
	}
	return result, nil
}

func parseEventReportLinkList(msg *ast.DataMessage) ([]eventReportLinkMessage, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}
	root, err := msg.Get()
	if err != nil {
		return nil, err
	}
	list, ok := root.(*ast.ListNode)
	if !ok {
		return nil, fmt.Errorf("expected list payload for S2F35")
	}

	if list.Size() == 0 {
		return nil, nil
	}

	var entries *ast.ListNode
	firstNode, err := list.Get(0)
	if err != nil {
		return nil, err
	}
	if _, ok := firstNode.(*ast.ListNode); ok {
		entries = list
	} else {
		if list.Size() < 2 {
			return nil, fmt.Errorf("malformed S2F35 payload")
		}
		secondNode, err := list.Get(1)
		if err != nil {
			return nil, err
		}
		entryList, ok := secondNode.(*ast.ListNode)
		if !ok {
			return nil, fmt.Errorf("malformed S2F35 link list")
		}
		entries = entryList
	}

	result := make([]eventReportLinkMessage, 0, entries.Size())
	for i := 0; i < entries.Size(); i++ {
		entryNode, err := entries.Get(i)
		if err != nil {
			return nil, err
		}
		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() < 2 {
			return nil, fmt.Errorf("malformed S2F35 entry")
		}
		ceNode, err := entry.Get(0)
		if err != nil {
			return nil, err
		}
		ceid, err := newIDInfoFromNode(ceNode)
		if err != nil {
			return nil, err
		}
		rptListNode, err := entry.Get(1)
		if err != nil {
			return nil, err
		}
		rptList, ok := rptListNode.(*ast.ListNode)
		if !ok {
			return nil, fmt.Errorf("malformed S2F35 RPTID list")
		}

		rptInfos := make([]idInfo, 0, rptList.Size())
		for idx := 0; idx < rptList.Size(); idx++ {
			rptNode, err := rptList.Get(idx)
			if err != nil {
				return nil, err
			}
			rptID, err := newIDInfoFromNode(rptNode)
			if err != nil {
				return nil, err
			}
			rptInfos = append(rptInfos, rptID)
		}

		result = append(result, eventReportLinkMessage{ceid: ceid, rptids: rptInfos})
	}

	return result, nil
}

func parseEventEnableMessage(msg *ast.DataMessage) (eventEnableMessage, error) {
	var result eventEnableMessage
	if msg == nil {
		return result, fmt.Errorf("nil message")
	}

	root, err := msg.Get()
	if err != nil {
		return result, err
	}
	list, ok := root.(*ast.ListNode)
	if !ok || list.Size() < 2 {
		return result, fmt.Errorf("malformed S2F37 payload")
	}

	ceedNode, err := list.Get(0)
	if err != nil {
		return result, err
	}
	switch node := ceedNode.(type) {
	case *ast.BinaryNode:
		values, ok := node.Values().([]int)
		if !ok || len(values) == 0 {
			return result, fmt.Errorf("invalid CEED payload")
		}
		result.enable = values[0] != 0
	case *ast.BooleanNode:
		values, ok := node.Values().([]bool)
		if !ok || len(values) == 0 {
			return result, fmt.Errorf("invalid CEED payload")
		}
		result.enable = values[0]
	default:
		return result, fmt.Errorf("expected boolean CEED")
	}

	ceListNode, err := list.Get(1)
	if err != nil {
		return result, err
	}
	ceList, ok := ceListNode.(*ast.ListNode)
	if !ok {
		return result, fmt.Errorf("malformed CEID list")
	}

	for i := 0; i < ceList.Size(); i++ {
		ceNode, err := ceList.Get(i)
		if err != nil {
			return result, err
		}
		info, err := newIDInfoFromNode(ceNode)
		if err != nil {
			return result, err
		}
		result.ceids = append(result.ceids, info)
	}

	return result, nil
}

func parseEventReportRequest(msg *ast.DataMessage) (idInfo, error) {
	if msg == nil {
		return idInfo{}, fmt.Errorf("nil message")
	}

	root, err := msg.Get()
	if err != nil {
		return idInfo{}, err
	}

	if list, ok := root.(*ast.ListNode); ok {
		if list.Size() == 0 {
			return idInfo{}, fmt.Errorf("empty S6F15 payload")
		}
		first, err := list.Get(0)
		if err != nil {
			return idInfo{}, err
		}
		return newIDInfoFromNode(first)
	}

	return newIDInfoFromNode(root)
}

func parseEventReportMessage(msg *ast.DataMessage) (EventReport, error) {
	var report EventReport
	if msg == nil {
		return report, fmt.Errorf("nil message")
	}

	root, err := msg.Get()
	if err != nil {
		return report, err
	}
	list, ok := root.(*ast.ListNode)
	if !ok || list.Size() < 3 {
		return report, fmt.Errorf("malformed S6F11 payload")
	}

	dataIDNode, err := list.Get(0)
	if err != nil {
		return report, err
	}
	switch node := dataIDNode.(type) {
	case *ast.BinaryNode:
		if vals, ok := node.Values().([]int); ok && len(vals) > 0 {
			report.DATAID = vals[0]
		}
	case *ast.UintNode:
		if vals, ok := node.Values().([]uint64); ok && len(vals) > 0 {
			report.DATAID = int(vals[0])
		}
	case *ast.IntNode:
		if vals, ok := node.Values().([]int64); ok && len(vals) > 0 {
			report.DATAID = int(vals[0])
		}
	}

	ceNode, err := list.Get(1)
	if err != nil {
		return report, err
	}
	ceInfo, err := newIDInfoFromNode(ceNode)
	if err != nil {
		return report, err
	}
	report.CEID = ceInfo.raw

	reportListNode, err := list.Get(2)
	if err != nil {
		return report, err
	}
	reportList, ok := reportListNode.(*ast.ListNode)
	if !ok {
		return report, fmt.Errorf("malformed S6F11 report list")
	}

	report.Reports = make([]ReportValue, 0, reportList.Size())
	for i := 0; i < reportList.Size(); i++ {
		entryNode, err := reportList.Get(i)
		if err != nil {
			return report, err
		}
		entry, ok := entryNode.(*ast.ListNode)
		if !ok || entry.Size() < 2 {
			return report, fmt.Errorf("malformed S6F11 report entry")
		}
		rptNode, err := entry.Get(0)
		if err != nil {
			return report, err
		}
		rptInfo, err := newIDInfoFromNode(rptNode)
		if err != nil {
			return report, err
		}
		valuesNode, err := entry.Get(1)
		if err != nil {
			return report, err
		}
		valuesList, ok := valuesNode.(*ast.ListNode)
		if !ok {
			return report, fmt.Errorf("malformed S6F11 value list")
		}

		values := make([]ast.ItemNode, 0, valuesList.Size())
		for idx := 0; idx < valuesList.Size(); idx++ {
			valueNode, err := valuesList.Get(idx)
			if err != nil {
				return report, err
			}
			values = append(values, valueNode)
		}

		report.Reports = append(report.Reports, ReportValue{RPTID: rptInfo.raw, Values: values})
	}

	return report, nil
}
