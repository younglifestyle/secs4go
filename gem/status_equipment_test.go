package gem_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/younglifestyle/secs4go/gem"
	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func TestStatusVariableAndEquipmentConstantFlows(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	port := 6500 + rand.Intn(500)

	equipmentProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, false, 0x0100, "equipment")
	hostProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, true, 0x0100, "host")

	equipmentProtocol.Timeouts().SetLinktest(60)
	hostProtocol.Timeouts().SetLinktest(60)

	equipmentHandler, err := gem.NewGemHandler(gem.Options{
		Protocol:   equipmentProtocol,
		DeviceType: gem.DeviceEquipment,
	})
	if err != nil {
		t.Fatalf("create equipment handler: %v", err)
	}

	hostHandler, err := gem.NewGemHandler(gem.Options{
		Protocol:   hostProtocol,
		DeviceType: gem.DeviceHost,
	})
	if err != nil {
		t.Fatalf("create host handler: %v", err)
	}

	defer equipmentHandler.Disable()
	defer hostHandler.Disable()

	statusValue := ast.NewUintNode(4, 25)
	statusVar, err := gem.NewStatusVariable(1001, "Temperature", "C",
		gem.WithStatusValueProvider(func() (ast.ItemNode, error) {
			return statusValue, nil
		}),
	)
	if err != nil {
		t.Fatalf("create status variable: %v", err)
	}
	if err := equipmentHandler.RegisterStatusVariable(statusVar); err != nil {
		t.Fatalf("register status variable: %v", err)
	}

	ecValue := ast.NewUintNode(2, 10)
	eqConstant, err := gem.NewEquipmentConstant(2001, "Delay", ast.NewUintNode(2, 10),
		gem.WithEquipmentConstantUnit("sec"),
		gem.WithEquipmentConstantMin(ast.NewUintNode(2, 5)),
		gem.WithEquipmentConstantMax(ast.NewUintNode(2, 30)),
		gem.WithEquipmentConstantValueProvider(func() (ast.ItemNode, error) {
			return ecValue, nil
		}),
		gem.WithEquipmentConstantValueUpdater(func(node ast.ItemNode) error {
			ecValue = node
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("create equipment constant: %v", err)
	}
	if err := equipmentHandler.RegisterEquipmentConstant(eqConstant); err != nil {
		t.Fatalf("register equipment constant: %v", err)
	}

	equipmentHandler.Enable()
	time.Sleep(200 * time.Millisecond)
	hostHandler.Enable()

	if !hostHandler.WaitForCommunicating(5 * time.Second) {
		t.Fatal("host failed to reach communicating state")
	}
	if !equipmentHandler.WaitForCommunicating(5 * time.Second) {
		t.Fatal("equipment failed to reach communicating state")
	}

	statusValues, err := hostHandler.RequestStatusVariables(uint16(1001))
	if err != nil {
		t.Fatalf("RequestStatusVariables: %v", err)
	}
	if len(statusValues) != 1 {
		t.Fatalf("expected 1 status value, got %d", len(statusValues))
	}
	assertUintValue(t, statusValues[0].Value, 25)

	// Update provider value and verify host sees the change.
	statusValue = ast.NewUintNode(4, 30)
	statusValues, err = hostHandler.RequestStatusVariables(uint16(1001))
	if err != nil {
		t.Fatalf("RequestStatusVariables: %v", err)
	}
	assertUintValue(t, statusValues[0].Value, 30)

	statusInfo, err := hostHandler.RequestStatusVariableInfo(uint16(1001))
	if err != nil {
		t.Fatalf("RequestStatusVariableInfo: %v", err)
	}
	if len(statusInfo) != 1 {
		t.Fatalf("expected 1 status info entry, got %d", len(statusInfo))
	}
	if statusInfo[0].Name != "Temperature" || statusInfo[0].Unit != "C" {
		t.Fatalf("unexpected info: %+v", statusInfo[0])
	}

	ecValues, err := hostHandler.RequestEquipmentConstants(uint16(2001))
	if err != nil {
		t.Fatalf("RequestEquipmentConstants: %v", err)
	}
	if len(ecValues) != 1 {
		t.Fatalf("expected 1 equipment constant value, got %d", len(ecValues))
	}
	assertUintValue(t, ecValues[0].Value, 10)

	ack, err := hostHandler.SendEquipmentConstantValues([]gem.EquipmentConstantUpdate{{
		ID:    uint16(2001),
		Value: ast.NewUintNode(2, 20),
	}})
	if err != nil {
		t.Fatalf("SendEquipmentConstantValues: %v", err)
	}
	if ack != 0 {
		t.Fatalf("unexpected EC ACK %d", ack)
	}

	ecValues, err = hostHandler.RequestEquipmentConstants(uint16(2001))
	if err != nil {
		t.Fatalf("RequestEquipmentConstants: %v", err)
	}
	assertUintValue(t, ecValues[0].Value, 20)

	ecInfo, err := hostHandler.RequestEquipmentConstantInfo(uint16(2001))
	if err != nil {
		t.Fatalf("RequestEquipmentConstantInfo: %v", err)
	}
	if len(ecInfo) != 1 {
		t.Fatalf("expected 1 equipment constant info, got %d", len(ecInfo))
	}
	if ecInfo[0].Name != "Delay" || ecInfo[0].Unit != "sec" {
		t.Fatalf("unexpected EC info: %+v", ecInfo[0])
	}
	assertUintValue(t, ecInfo[0].Min, 5)
	assertUintValue(t, ecInfo[0].Max, 30)
	assertUintValue(t, ecInfo[0].Default, 10)
}

func assertUintValue(t *testing.T, node ast.ItemNode, want int) {
	t.Helper()
	if node == nil {
		t.Fatalf("expected value %d but node is nil", want)
	}
	switch typed := node.(type) {
	case *ast.UintNode:
		values, ok := typed.Values().([]uint64)
		if !ok || len(values) == 0 {
			t.Fatalf("unexpected uint node payload: %v", typed)
		}
		if int(values[0]) != want {
			t.Fatalf("unexpected uint value %d (want %d)", values[0], want)
		}
	case *ast.IntNode:
		values, ok := typed.Values().([]int64)
		if !ok || len(values) == 0 {
			t.Fatalf("unexpected int node payload: %v", typed)
		}
		if int(values[0]) != want {
			t.Fatalf("unexpected int value %d (want %d)", values[0], want)
		}
	default:
		t.Fatalf("unexpected node type %T", node)
	}
}

func TestCollectionEventsAndProcessPrograms(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	port := 7100 + rand.Intn(500)

	equipmentProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, false, 0x0200, "equipment")
	hostProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, true, 0x0200, "host")

	equipmentProtocol.Timeouts().SetLinktest(60)
	hostProtocol.Timeouts().SetLinktest(60)

	equipmentHandler, err := gem.NewGemHandler(gem.Options{
		Protocol:   equipmentProtocol,
		DeviceType: gem.DeviceEquipment,
	})
	if err != nil {
		t.Fatalf("create equipment handler: %v", err)
	}

	hostHandler, err := gem.NewGemHandler(gem.Options{
		Protocol:   hostProtocol,
		DeviceType: gem.DeviceHost,
	})
	if err != nil {
		t.Fatalf("create host handler: %v", err)
	}

	defer equipmentHandler.Disable()
	defer hostHandler.Disable()

	statusValue := 100
	statusVar, err := gem.NewStatusVariable(1101, "StatusValue", "", gem.WithStatusValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(4, statusValue), nil
	}))
	if err != nil {
		t.Fatalf("create status variable: %v", err)
	}
	if err := equipmentHandler.RegisterStatusVariable(statusVar); err != nil {
		t.Fatalf("register status variable: %v", err)
	}

	dataValue := 7
	dataVar, err := gem.NewDataVariable(2101, "DataValue", gem.WithDataValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(2, dataValue), nil
	}))
	if err != nil {
		t.Fatalf("create data variable: %v", err)
	}
	if err := equipmentHandler.RegisterDataVariable(dataVar); err != nil {
		t.Fatalf("register data variable: %v", err)
	}

	collectionEvent, err := gem.NewCollectionEvent(3101, "TestEvent")
	if err != nil {
		t.Fatalf("create collection event: %v", err)
	}
	if err := equipmentHandler.RegisterCollectionEvent(collectionEvent); err != nil {
		t.Fatalf("register collection event: %v", err)
	}

	equipmentHandler.Enable()
	time.Sleep(200 * time.Millisecond)
	hostHandler.Enable()

	if !hostHandler.WaitForCommunicating(5 * time.Second) {
		t.Fatal("host failed to reach communicating state")
	}
	if !equipmentHandler.WaitForCommunicating(5 * time.Second) {
		t.Fatal("equipment failed to reach communicating state")
	}

	reportDef := gem.ReportDefinitionRequest{ReportID: 4001, VIDs: []interface{}{1101, 2101}}
	if ack, err := hostHandler.DefineReports(reportDef); err != nil {
		t.Fatalf("DefineReports error: %v", err)
	} else if ack != 0 {
		t.Fatalf("unexpected DefineReports ack %d", ack)
	}

	linkReq := gem.EventReportLinkRequest{CEID: 3101, ReportIDs: []interface{}{4001}}
	if ack, err := hostHandler.LinkEventReports(linkReq); err != nil {
		t.Fatalf("LinkEventReports error: %v", err)
	} else if ack != 0 {
		t.Fatalf("unexpected LinkEventReports ack %d", ack)
	}

	if ack, err := hostHandler.EnableEventReports(true, 3101); err != nil {
		t.Fatalf("EnableEventReports error: %v", err)
	} else if ack != 0 {
		t.Fatalf("unexpected EnableEventReports ack %d", ack)
	}

	eventCh := make(chan gem.EventReport, 1)
	hostHandler.Events().EventReportReceived.AddCallback(func(data map[string]interface{}) {
		if rpt, ok := data["report"].(gem.EventReport); ok {
			select {
			case eventCh <- rpt:
			default:
			}
		}
	})

	if err := equipmentHandler.TriggerCollectionEvent(3101); err != nil {
		t.Fatalf("TriggerCollectionEvent: %v", err)
	}

	select {
	case rpt := <-eventCh:
		if len(rpt.Reports) != 1 {
			t.Fatalf("expected 1 report, got %d", len(rpt.Reports))
		}
		if got := len(rpt.Reports[0].Values); got != 2 {
			t.Fatalf("expected 2 VID values, got %d", got)
		}
		assertUintValue(t, rpt.Reports[0].Values[0], statusValue)
		assertUintValue(t, rpt.Reports[0].Values[1], dataValue)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event report")
	}

	statusValue = 150
	dataValue = 9
	if err := equipmentHandler.TriggerCollectionEvent(3101); err != nil {
		t.Fatalf("TriggerCollectionEvent: %v", err)
	}

	select {
	case rpt := <-eventCh:
		assertUintValue(t, rpt.Reports[0].Values[0], statusValue)
		assertUintValue(t, rpt.Reports[0].Values[1], dataValue)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for updated event report")
	}

	rpt, err := hostHandler.RequestCollectionEventReport(3101)
	if err != nil {
		t.Fatalf("RequestCollectionEventReport: %v", err)
	}
	if len(rpt.Reports) == 0 {
		t.Fatalf("expected report data in S6F16 response")
	}
	assertUintValue(t, rpt.Reports[0].Values[0], statusValue)
	assertUintValue(t, rpt.Reports[0].Values[1], dataValue)

	if ack, err := hostHandler.UploadProcessProgram("PP1", "BODY-1"); err != nil {
		t.Fatalf("UploadProcessProgram: %v", err)
	} else if ack != 0 {
		t.Fatalf("unexpected process program ack %d", ack)
	}

	programs := equipmentHandler.ListProcessPrograms()
	if len(programs) != 1 || programs[0].Body != "BODY-1" {
		t.Fatalf("unexpected stored process programs: %+v", programs)
	}

	body, ack, err := hostHandler.RequestProcessProgram("PP1")
	if err != nil {
		t.Fatalf("RequestProcessProgram: %v", err)
	}
	if ack != 0 {
		t.Fatalf("unexpected process program request ack %d", ack)
	}
	if body != "BODY-1" {
		t.Fatalf("unexpected process program body %q", body)
	}

	_, missingAck, err := hostHandler.RequestProcessProgram("UNKNOWN")
	if err != nil {
		t.Fatalf("RequestProcessProgram missing: %v", err)
	}
	if missingAck == 0 {
		t.Fatalf("expected non-zero ack for unknown process program")
	}
}
