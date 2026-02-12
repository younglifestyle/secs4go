package main

import (
	"flag"
	"fmt"
	"github.com/spf13/cast"
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
	addr := flag.String("addr", "127.0.0.1", "Equipment address")
	port := flag.Int("port", 15000, "Equipment port")
	session := flag.Int("session", 2, "HSMS session identifier")
	scenario := flag.String("scenario", "all", "Operation scenario: status|reports|remote|process|all")
	flag.Parse()

	protocol := hsms.NewHsmsProtocol(*addr, *port, true, *session, "example2-host")
	protocol.Timeouts().SetLinktest(30)
	protocol.Timeouts().SetT8NetworkIntercharTimeout(45)

	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceHost,
		Logging: gem.LoggingOptions{
			Enabled:                true,
			Mode:                   hsms.LoggingModeSML,
			IncludeControlMessages: true,
		},
	})
	if err != nil {
		log.Fatalf("create GEM handler: %v", err)
	}

	eventCh := make(chan gem.EventReport, 4)
	installHostCallbacks(handler, eventCh)

	handler.Enable()
	defer handler.Disable()

	log.Printf("connecting to equipment at %s:%d session=0x%X", *addr, *port, *session)
	if !handler.WaitForCommunicating(60 * time.Second) {
		log.Fatal("timeout waiting for COMMUNICATING state")
	}
	log.Println("host in COMMUNICATING state")

	scen := strings.ToLower(*scenario)
	switch scen {
	case "status":
		runStatusPolling(handler)
	case "reports":
		runReportWorkflow(handler, eventCh)
	case "remote":
		runRemoteCommand(handler, eventCh)
	case "process":
		runProcessProgram(handler)
	case "all":
		runStatusPolling(handler)
		runReportWorkflow(handler, eventCh)
		runRemoteCommand(handler, eventCh)
		runProcessProgram(handler)
	default:
		log.Fatalf("unknown scenario %q", scen)
	}

	log.Println("operations complete; waiting for Ctrl+C to disconnect")
	<-interrupt()
}

func installHostCallbacks(handler *gem.GemHandler, eventCh chan<- gem.EventReport) {
	handler.Events().ControlStateChanged.AddCallback(func(data map[string]interface{}) {
		prev, _ := data["previous"].(gem.ControlState)
		next, _ := data["current"].(gem.ControlState)
		mode, _ := data["mode"].(gem.OnlineControlMode)
		log.Printf("control state changed: %s -> %s (mode=%s)", prev, next, mode)
	})

	handler.Events().EventReportReceived.AddCallback(func(data map[string]interface{}) {
		if rpt, ok := data["report"].(gem.EventReport); ok {
			log.Printf("S6F11 received CEID=%v DATAID=%d REPORTS=%d", rpt.CEID, rpt.DATAID, len(rpt.Reports))
			select {
			case eventCh <- rpt:
			default:
			}
		}
	})

	handler.Events().S9ErrorReceived.AddCallback(func(data map[string]interface{}) {
		if info, ok := data["error"].(*hsms.S9ErrorInfo); ok {
			log.Printf("S9 Error Received: Stream=%d Function=%d Text=%s Data=%X", 
				info.ErrorCode, info.ErrorCode, info.ErrorText, info.SystemBytes)
		}
	})
}

//func runStatusPolling(handler *gem.GemHandler) {
//	log.Println("--- STATUS VARIABLE SNAPSHOT ---")
//	infos, err := handler.RequestStatusVariableInfo()
//	if err != nil {
//		log.Printf("RequestStatusVariableInfo error: %v", err)
//		return
//	}
//	if len(infos) == 0 {
//		log.Println("equipment reported no status variables")
//		return
//	}
//	ids := make([]interface{}, 0, len(infos))
//	for _, info := range infos {
//		log.Printf("SVID=%v NAME=%s UNIT=%s", info.ID, info.Name, info.Unit)
//		// ✅ 跳过保留 SVID（1001~1005），避免触发设备端特殊分支
//		if n, ok := info.ID.(int64); ok {
//			if n >= 1001 && n <= 1005 {
//				continue
//			}
//		}
//		ids = append(ids, info.ID)
//	}
//
//	if len(ids) == 0 {
//		log.Println("no non-reserved SVIDs to request")
//		return
//	}
//
//	values, err := handler.RequestStatusVariables(ids...)
//	if err != nil {
//		log.Printf("RequestStatusVariables error: %v", err)
//		return
//	}
//	for _, value := range values {
//		log.Printf("SVID=%v VALUE=%s", value.ID, describeItem(value.Value))
//	}
//}

//func runStatusPolling(handler *gem.GemHandler) {
//	log.Println("--- STATUS VARIABLE SNAPSHOT ---")
//	values, err := handler.RequestStatusVariables(int64(1101), int64(2001), int64(3002))
//	if err != nil {
//		log.Printf("RequestStatusVariables error: %v", err)
//		return
//	}
//	for _, v := range values {
//		log.Printf("SVID=%v VALUE=%s", v.ID, describeItem(v.Value))
//	}
//}

func runStatusPolling(handler *gem.GemHandler) {
	log.Println("--- STATUS VARIABLE SNAPSHOT ---")

	infos, err := handler.RequestStatusVariableInfo()
	if err != nil {
		log.Printf("RequestStatusVariableInfo error: %v", err)
		return
	}
	if len(infos) == 0 {
		log.Println("equipment reported no status variables")
		return
	}

	ids := make([]interface{}, 0, len(infos))
	for _, info := range infos {
		log.Printf("SVID=%v NAME=%s UNIT=%s", info.ID, info.Name, info.Unit)

		// 过滤保留 SVID（1001..1005）
		if n := cast.ToInt(info.ID); n >= 1001 && n <= 1005 {
			continue
		}

		// 只请求我们关心的（也可以放行全部非保留项）
		if n, ok := info.ID.(int64); ok {
			if n != 1101 && n != 2001 && n != 3002 {
				continue
			}
		}
		ids = append(ids, info.ID)
	}

	if len(ids) == 0 {
		log.Println("no non-reserved SVIDs to request")
		return
	}

	fmt.Println("ids : ", ids)

	values, err := handler.RequestStatusVariables(ids...)
	if err != nil {
		log.Printf("RequestStatusVariables error: %v", err)
		return
	}
	for _, v := range values {
		log.Printf("SVID=%v VALUE=%s", v.ID, describeItem(v.Value))
	}
}

func runReportWorkflow(handler *gem.GemHandler, eventCh <-chan gem.EventReport) {
	log.Println("--- REPORT DEFINITION WORKFLOW ---")
	if ack, err := handler.DefineReports(); err != nil {
		log.Printf("DefineReports(clear) error: %v", err)
		return
	} else if ack != 0 {
		log.Printf("DefineReports(clear) returned DRACK=%d", ack)
	}

	report := gem.ReportDefinitionRequest{ReportID: 4001, VIDs: []interface{}{int64(1101), int64(2001)}}
	if ack, err := handler.DefineReports(report); err != nil {
		log.Printf("DefineReports error: %v", err)
		return
	} else if ack != 0 {
		log.Printf("DefineReports returned DRACK=%d", ack)
	}

	if ack, err := handler.LinkEventReports(gem.EventReportLinkRequest{CEID: int64(3001), ReportIDs: []interface{}{report.ReportID}}); err != nil {
		log.Printf("LinkEventReports error: %v", err)
	} else if ack != 0 {
		log.Printf("LinkEventReports returned LRACK=%d", ack)
	}

	if ack, err := handler.EnableEventReports(true, int64(3001)); err != nil {
		log.Printf("EnableEventReports error: %v", err)
	} else if ack != 0 {
		log.Printf("EnableEventReports returned ERACK=%d", ack)
	}

	log.Println("waiting up to 10s for an S6F11 report ...")
	select {
	case rpt := <-eventCh:
		log.Printf("received CEID=%v with %d reports", rpt.CEID, len(rpt.Reports))
	case <-time.After(10 * time.Second):
		log.Println("no S6F11 received within 10 seconds")
	}

	if snapshot, err := handler.RequestCollectionEventReport(int64(3001)); err != nil {
		log.Printf("RequestCollectionEventReport error: %v", err)
	} else {
		log.Printf("S6F16 snapshot CEID=%v REPORTS=%d", snapshot.CEID, len(snapshot.Reports))
	}
}

func runRemoteCommand(handler *gem.GemHandler, eventCh <-chan gem.EventReport) {
	log.Println("--- REMOTE COMMAND START ---")
	result, err := handler.SendRemoteCommand("START", nil)
	if err != nil {
		log.Printf("SendRemoteCommand error: %v", err)
		return
	}
	log.Printf("remote command START HCACK=%d", result.HCACK)
	if result.HCACK != gem.HCACKAcknowledge && result.HCACK != gem.HCACKAcknowledgeLater {
		log.Printf("remote command not accepted (HCACK=%d); skipping", result.HCACK)
		return
	}

	select {
	case rpt := <-eventCh:
		log.Printf("remote command triggered CEID=%v", rpt.CEID)
	case <-time.After(5 * time.Second):
		log.Println("no event received after remote command")
	}
}

func runProcessProgram(handler *gem.GemHandler) {
	log.Println("--- PROCESS PROGRAM ROUND-TRIP ---")
	if ack, err := handler.UploadProcessProgram("SAMPLE", "GDSCRIPT-001"); err != nil {
		log.Printf("UploadProcessProgram error: %v", err)
		return
	} else {
		log.Printf("upload ack=%d", ack)
	}

	if body, ack, err := handler.RequestProcessProgram("SAMPLE"); err != nil {
		log.Printf("RequestProcessProgram error: %v", err)
		return
	} else {
		log.Printf("download ack=%d body=%q", ack, body)
	}
}

func describeItem(node ast.ItemNode) string {
	if node == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", node)
}

func interrupt() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
