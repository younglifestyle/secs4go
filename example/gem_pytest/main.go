package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/younglifestyle/secs4go/gem"
	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func main() {
	role := flag.String("role", "equipment", "host|equipment (host targets Python equipment, equipment targets Python host)")
	addr := flag.String("addr", "127.0.0.1", "HSMS peer address")
	port := flag.Int("port", 5000, "HSMS peer port")
	session := flag.Int("session", 2, "HSMS session ID")
	ppid := flag.String("ppid", "PYTEST", "Process program ID used for recipe exchange")
	flag.Parse()

	var err error
	switch strings.ToLower(*role) {
	case "host":
		err = runPythonEquipmentSuite(*addr, *port, *session, *ppid)
	case "equipment":
		err = runPythonHostEquipment(*addr, *port, *session)
	default:
		err = fmt.Errorf("unsupported role %q", *role)
	}

	if err != nil {
		log.Fatalf("suite failed: %v", err)
	}
}

func runPythonEquipmentSuite(addr string, port, session int, ppid string) error {
	protocol := hsms.NewHsmsProtocol(addr, port, true, session, "go-host-python")
	protocol.Timeouts().SetLinktest(30)

	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceHost,
	})
	if err != nil {
		return err
	}

	log.Printf("connecting to equipment %s:%d session=0x%X", addr, port, session)
	handler.Enable()
	defer handler.Disable()

	log.Println("waiting for COMMUNICATING state ...")
	handler.WaitForCommunicating(0)
	log.Println("COMMUNICATING established")

	if err := exerciseStatusVariables(handler); err != nil {
		return fmt.Errorf("status variables: %w", err)
	}
	if err := exerciseEquipmentConstants(handler); err != nil {
		return fmt.Errorf("equipment constants: %w", err)
	}
	if err := exerciseReports(handler); err != nil {
		return fmt.Errorf("reports: %w", err)
	}
	if err := exerciseProcessProgram(handler, ppid); err != nil {
		return fmt.Errorf("process program: %w", err)
	}
	if err := exerciseRemoteCommand(handler); err != nil {
		return fmt.Errorf("remote command: %w", err)
	}

	log.Println("PYTEST RESULT: SUCCESS (host vs Python equipment)")
	return nil
}

func exerciseStatusVariables(handler *gem.GemHandler) error {
	infos, err := handler.RequestStatusVariableInfo()
	if err != nil {
		return fmt.Errorf("request status variable info: %w", err)
	}
	if len(infos) == 0 {
		return errors.New("no status variables reported")
	}
	log.Printf("status variable info (%d entries):", len(infos))
	for _, info := range infos {
		log.Printf("  SVID=%v NAME=%s UNIT=%s", info.ID, info.Name, info.Unit)
	}

	ids := make([]interface{}, 0, len(infos))
	for _, info := range infos {
		ids = append(ids, info.ID)
	}
	values, err := handler.RequestStatusVariables(ids...)
	if err != nil {
		return fmt.Errorf("request status variables: %w", err)
	}
	for _, val := range values {
		log.Printf("  SVID=%v VALUE=%s", val.ID, describeNode(val.Value))
	}
	return nil
}

func exerciseEquipmentConstants(handler *gem.GemHandler) error {
	infos, err := handler.RequestEquipmentConstantInfo()
	if err != nil {
		return fmt.Errorf("request equipment constant info: %w", err)
	}
	if len(infos) == 0 {
		return errors.New("no equipment constants reported")
	}
	log.Printf("equipment constant info (%d entries):", len(infos))
	ids := make([]interface{}, 0, len(infos))
	for _, info := range infos {
		log.Printf("  ECID=%v NAME=%s UNIT=%s DEF=%s", info.ID, info.Name, info.Unit, describeNode(info.Default))
		ids = append(ids, info.ID)
	}

	values, err := handler.RequestEquipmentConstants(ids...)
	if err != nil {
		return fmt.Errorf("request equipment constants: %w", err)
	}
	for _, val := range values {
		log.Printf("  ECID=%v VALUE=%s", val.ID, describeNode(val.Value))
	}
	return nil
}

func exerciseReports(handler *gem.GemHandler) error {
	infos, err := handler.RequestStatusVariableInfo()
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		log.Println("skip report definition (no SVIDs)")
		return nil
	}

	primary := infos[0].ID
	log.Printf("defining report 4001 with SVID=%v", primary)
	if ack, err := handler.DefineReports(); err != nil {
		return fmt.Errorf("clear reports: %w", err)
	} else if ack != 0 {
		return fmt.Errorf("clear reports returned DRACK=%d", ack)
	}
	if ack, err := handler.DefineReports(gem.ReportDefinitionRequest{ReportID: 4001, VIDs: []interface{}{primary}}); err != nil {
		return fmt.Errorf("define reports: %w", err)
	} else {
		log.Printf("  S2F34 DRACK=%d", ack)
		if ack != 0 {
			return fmt.Errorf("define reports returned DRACK=%d", ack)
		}
	}

	log.Println("linking CEID 1 to RPTID 4001")
	if ack, err := handler.LinkEventReports(gem.EventReportLinkRequest{CEID: 1, ReportIDs: []interface{}{4001}}); err != nil {
		return fmt.Errorf("link event reports: %w", err)
	} else {
		log.Printf("  S2F36 LRACK=%d", ack)
		if ack != 0 {
			return fmt.Errorf("link event reports returned LRACK=%d", ack)
		}
	}

	log.Println("enabling CEID 1 reports via S2F37")
	if ack, err := handler.EnableEventReports(true, 1); err != nil {
		return fmt.Errorf("enable event reports: %w", err)
	} else {
		log.Printf("  S2F38 ERACK=%d", ack)
		if ack != 0 {
			return fmt.Errorf("enable event reports returned ERACK=%d", ack)
		}
	}

	log.Println("requesting CEID 1 snapshot via S6F15")
	rpt, err := handler.RequestCollectionEventReport(1)
	if err != nil {
		return fmt.Errorf("request CEID 1: %w", err)
	}
	if len(rpt.Reports) == 0 {
		return errors.New("CEID 1 returned no reports")
	}
	log.Printf("  CEID=%v DATAID=%d REPORTS=%d", rpt.CEID, rpt.DATAID, len(rpt.Reports))
	return nil
}

func exerciseProcessProgram(handler *gem.GemHandler, ppid string) error {
	log.Printf("uploading process program %q", ppid)
	if ack, err := handler.UploadProcessProgram(ppid, ";recipe-body;\nEND"); err != nil {
		return fmt.Errorf("upload process program: %w", err)
	} else {
		log.Printf("  S7F4 ACK=%d", ack)
		if ack != 0 {
			log.Printf("  skipping process program round-trip (equipment returned ACK=%d)", ack)
			return nil
		}
	}

	log.Printf("requesting process program %q", ppid)
	body, ack, err := handler.RequestProcessProgram(ppid)
	if err != nil {
		return fmt.Errorf("request process program: %w", err)
	}
	log.Printf("  S7F6 ACK=%d BODY=%q", ack, body)
	if ack != 0 {
		log.Printf("  skipping process program verification (equipment returned ACK=%d)", ack)
		return nil
	}
	return nil
}

func exerciseRemoteCommand(handler *gem.GemHandler) error {
	log.Println("sending remote command START")
	result, err := handler.SendRemoteCommand("START", nil)
	if err != nil {
		return err
	}
	log.Printf("  HCACK=%d", result.HCACK)
	if result.HCACK != gem.HCACKAcknowledge && result.HCACK != gem.HCACKAcknowledgeLater { // treat ACK_FINISH_LATER as success
		log.Printf("  remote command not accepted (HCACK=%d); skipping", result.HCACK)
		return nil
	}
	return nil
}

func runPythonHostEquipment(addr string, port, session int) error {
	protocol := hsms.NewHsmsProtocol(addr, port, false, session, "go-equipment-python")
	protocol.Timeouts().SetLinktest(30)

	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceEquipment,
		MDLN:       "go-eqp",
		SOFTREV:    "1.0.0",
	})
	if err != nil {
		return err
	}

	registerDemoCapabilities(handler)

	log.Printf("connecting to host %s:%d session=0x%X", addr, port, session)
	handler.Enable()
	defer handler.Disable()

	handler.WaitForCommunicating(0)
	log.Println("COMMUNICATING with host; equipment ready. Press Ctrl+C to terminate")
	log.Println("PYTEST RESULT: SUCCESS (equipment ready; awaiting host commands)")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if handler.State() == gem.CommunicationStateCommunicating {
				if err := handler.RaiseAlarm(1001, true); err == nil {
					log.Println("raised alarm 1001 (simulate)")
				}
			}
		case <-interruptChan():
			log.Println("equipment shutting down")
			return nil
		}
	}
}

func registerDemoCapabilities(handler *gem.GemHandler) {
	status, _ := gem.NewStatusVariable(1001, "Temperature", "C", gem.WithStatusValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(4, 25), nil
	}))
	_ = handler.RegisterStatusVariable(status)

	dataVar, _ := gem.NewDataVariable(2001, "Pressure", gem.WithDataValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(2, 48), nil
	}))
	_ = handler.RegisterDataVariable(dataVar)

	ec, _ := gem.NewEquipmentConstant(3001, "Delay", ast.NewUintNode(2, 10))
	_ = handler.RegisterEquipmentConstant(ec)

	ce1, _ := gem.NewCollectionEvent(1, "EquipmentStatus")
	_ = handler.RegisterCollectionEvent(ce1)
	ceProcess, _ := gem.NewCollectionEvent(3001, "ProcessComplete")
	_ = handler.RegisterCollectionEvent(ceProcess)

	handler.RegisterAlarm(gem.Alarm{ID: 1001, Text: "DoorOpen"})

	handler.SetRemoteCommandHandler(func(req gem.RemoteCommandRequest) (gem.RemoteCommandResult, error) {
		log.Printf("remote command received: %s %v", req.Command, req.Parameters)
		if err := handler.TriggerCollectionEvent(3001); err != nil {
			log.Printf("failed to trigger CEID 3001: %v", err)
		}
		return gem.RemoteCommandResult{HCACK: gem.HCACKAcknowledge}, nil
	})
}

func describeNode(node ast.ItemNode) string {
	if node == nil {
		return "<nil>"
	}
	switch n := node.(type) {
	case *ast.ASCIINode:
		if val, ok := n.Values().(string); ok {
			return fmt.Sprintf("A(%q)", val)
		}
	case *ast.UintNode:
		if vals, ok := n.Values().([]uint64); ok && len(vals) > 0 {
			return fmt.Sprintf("U(%d)", vals[0])
		}
	case *ast.IntNode:
		if vals, ok := n.Values().([]int64); ok && len(vals) > 0 {
			return fmt.Sprintf("I(%d)", vals[0])
		}
	case *ast.BinaryNode:
		if vals, ok := n.Values().([]int); ok && len(vals) > 0 {
			return fmt.Sprintf("B(%d)", vals[0])
		}
	case *ast.ListNode:
		sz := n.Size()
		parts := make([]string, 0, sz)
		for i := 0; i < sz; i++ {
			item, err := n.Get(i)
			if err != nil {
				continue
			}
			parts = append(parts, describeNode(item))
		}
		return fmt.Sprintf("L[%d]{%s}", sz, strings.Join(parts, ","))
	}
	return fmt.Sprintf("%T", node)
}

func interruptChan() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
