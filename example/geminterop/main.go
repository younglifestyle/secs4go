package main

import (
	"flag"
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
	mode := flag.String("mode", "host", "Mode to run: host or equipment")
	address := flag.String("addr", "127.0.0.1", "Remote HSMS address")
	port := flag.Int("port", 5000, "Remote HSMS port")
	session := flag.Int("session", 0x100, "HSMS session ID")
	flag.Parse()

	switch strings.ToLower(*mode) {
	case "host":
		runHost(*address, *port, *session)
	case "equipment":
		runEquipment(*address, *port, *session)
	default:
		log.Fatalf("unsupported mode %q", *mode)
	}
}

func runHost(address string, port, session int) {
	protocol := hsms.NewHsmsProtocol(address, port, true, session, "go-host")
	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceHost,
	})
	if err != nil {
		log.Fatalf("create host handler: %v", err)
	}

	handler.Events().EventReportReceived.AddCallback(func(data map[string]interface{}) {
		if rpt, ok := data["report"].(gem.EventReport); ok {
			log.Printf("S6F11 received: CEID=%v RPTs=%d", rpt.CEID, len(rpt.Reports))
		}
	})

	handler.Enable()
	defer handler.Disable()

	log.Println("Host waiting for equipment to reach COMMUNICATING ...")
	handler.WaitForCommunicating(0)
	log.Println("Host in COMMUNICATING state")

	log.Println("Resetting previous report definitions ...")
	if ack, err := handler.DefineReports(); err != nil {
		log.Fatalf("DefineReports (clear): %v", err)
	} else if ack != 0 {
		log.Printf("DefineReports (clear) returned DRACK=%d", ack)
	}

	log.Println("Connected. Defining report set ...")
	if ack, err := handler.DefineReports(gem.ReportDefinitionRequest{
		ReportID: 4001,
		VIDs:     []interface{}{int64(1001), int64(2001)},
	}); err != nil {
		log.Fatalf("DefineReports: %v", err)
	} else if ack != 0 {
		log.Fatalf("DefineReports returned DRACK=%d", ack)
	}

	log.Println("Linking event report ...")
	if ack, err := handler.LinkEventReports(gem.EventReportLinkRequest{
		CEID:      int64(3001),
		ReportIDs: []interface{}{int64(4001)},
	}); err != nil {
		log.Fatalf("LinkEventReports: %v", err)
	} else if ack != 0 {
		log.Fatalf("LinkEventReports returned LRACK=%d", ack)
	}

	log.Println("Enabling event reports ...")
	if ack, err := handler.EnableEventReports(true, int64(3001)); err != nil {
		log.Fatalf("EnableEventReports: %v", err)
	} else if ack != 0 {
		log.Fatalf("EnableEventReports returned ERACK=%d", ack)
	}

	log.Println("Uploading sample process program ...")
	if ack, err := handler.UploadProcessProgram("SAMPLE", "GDSCRIPT-001"); err != nil {
		log.Fatalf("UploadProcessProgram: %v", err)
	} else if ack != 0 {
		log.Printf("UploadProcessProgram returned ack=%d", ack)
	}

	log.Println("Requesting process program back ...")
	if body, ack, err := handler.RequestProcessProgram("SAMPLE"); err != nil {
		log.Printf("RequestProcessProgram error: %v", err)
	} else {
		log.Printf("Process program ack=%d body=%q", ack, body)
	}

	log.Println("Waiting for S6F11 notifications. Press Ctrl+C to exit.")
	waitForInterrupt()
}

func runEquipment(address string, port, session int) {
	protocol := hsms.NewHsmsProtocol(address, port, false, session, "go-equipment")
	protocol.Timeouts().SetLinktest(60)
	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceEquipment,
	})
	if err != nil {
		log.Fatalf("create equipment handler: %v", err)
	}

	statusVar, _ := gem.NewStatusVariable(1001, "Temperature", "C", gem.WithStatusValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(4, 23), nil
	}))
	if err := handler.RegisterStatusVariable(statusVar); err != nil {
		log.Fatalf("register status variable: %v", err)
	}

	dataVar, _ := gem.NewDataVariable(2001, "Pressure", gem.WithDataValueProvider(func() (ast.ItemNode, error) {
		return ast.NewUintNode(2, 42), nil
	}))
	if err := handler.RegisterDataVariable(dataVar); err != nil {
		log.Fatalf("register data variable: %v", err)
	}

	collectionEvent, _ := gem.NewCollectionEvent(3001, "StatusSnapshot")
	if err := handler.RegisterCollectionEvent(collectionEvent); err != nil {
		log.Fatalf("register collection event: %v", err)
	}

	if err := handler.RegisterProcessProgram("SAMPLE", "GDSCRIPT-001"); err != nil {
		log.Printf("seed process program: %v", err)
	}

	handler.Enable()
	defer handler.Disable()

	log.Println("Equipment waiting for host to establish communications ...")
	handler.WaitForCommunicating(0)
	log.Println("Equipment in COMMUNICATING state")

	log.Println("Equipment ready. Triggering CEID 3001 every 10s if linked.")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if handler.State() != gem.CommunicationStateCommunicating {
				continue
			}
			if err := handler.TriggerCollectionEvent(int64(3001)); err != nil {
				log.Printf("trigger collection event: %v", err)
			}
		case <-interrupt():
			log.Println("Shutting down equipment")
			return
		}
	}
}

func waitForInterrupt() {
	<-interrupt()
}

func interrupt() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
