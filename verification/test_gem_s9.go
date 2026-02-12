package main

import (
	"log"
	"time"

	"github.com/younglifestyle/secs4go/gem"
	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func main() {
	log.Println("Starting GEM S9 Error Verification...")

	// 1. Create GEM Handler (Host)
	// Use port 5020 to match sim_timeout.py
	protocol := hsms.NewHsmsProtocol("127.0.0.1", 5020, true, 10, "GemHost")

	opts := gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceHost,
		DeviceID:   10,
	}

	handler, err := gem.NewGemHandler(opts)
	if err != nil {
		log.Fatalf("Failed to create GEM handler: %v", err)
	}

	// 2. Subscribe to S9ErrorReceived event
	s9Received := make(chan *hsms.S9ErrorInfo, 1)
	handler.Events().S9ErrorReceived.AddCallback(func(data map[string]interface{}) {
		if val, ok := data["error"]; ok {
			if info, ok := val.(*hsms.S9ErrorInfo); ok {
				log.Printf("✓ Received S9 Error Event: Stream=%d Function=%d Text=%s",
					info.ErrorCode, info.ErrorCode, info.ErrorText)
				s9Received <- info
			}
		}
	})

	handler.Enable()
	defer handler.Disable()

	log.Println("GEM Handler enabled. Waiting for connection...")
	time.Sleep(3 * time.Second)

	// 3. Send Valid Message (S1F1) to trigger unsolicited S9F3 from Equipment
	log.Println("Sending S1F1 (Are You There) to trigger response + unsolicited S9F3...")

	s1f1 := ast.NewDataMessage("AreYouThere", 1, 1, 1, "H->E", ast.NewListNode())

	if err := handler.Protocol().SendDataMessage(s1f1); err != nil {
		log.Fatalf("Failed to send S1F1: %v", err)
	}

	// 4. Wait for S9 Error Event
	select {
	case info := <-s9Received:
		if info.ErrorCode == 3 { // S9F3 = Unrecognized Stream
			log.Println("SUCCESS: Correctly received S9F3 event")
		} else {
			log.Printf("FAILURE: Received unexpected S9 error code: %d (Expected 3)", info.ErrorCode)
		}
	case <-time.After(5 * time.Second):
		log.Println("FAILURE: Timed out waiting for S9 error event")
	}
}
