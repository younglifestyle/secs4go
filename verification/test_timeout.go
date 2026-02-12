package main

import (
	"log"
	"time"

	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func main() {
	log.Println("Starting Go Host for Timeout Test...")

	// Connect to 5020
	host := hsms.NewHsmsProtocol("127.0.0.1", 5020, true, 10, "GoHostTimeout")

	// Set short T3 timeout (2 seconds) to trigger S9F9 quickly
	// Equipment will delay response by 5 seconds
	host.Timeouts().SetT3ReplyTimeout(2)
	host.Timeouts().SetAutoReconnect(true)

	// Register generic handler for S9F9 to log it (though protocol handles sending it)
	// We want to see if we receive any S9 from equipment? No, we send S9 to equipment.
	// Equipment verifies receipt.

	// Handler for S1F13 to establish comms
	host.RegisterHandler(1, 13, func(msg *ast.DataMessage) (*ast.DataMessage, error) {
		log.Println("Received S1F13, accepting...")
		commack := byte(0)
		body := ast.NewListNode(ast.NewBinaryNode(commack), ast.NewListNode())
		return ast.NewDataMessage("EstablishCommAck", 1, 14, 0, "H->E", body), nil
	})

	host.Enable()

	log.Println("Host enabled. Waiting 2 seconds for connection...")
	time.Sleep(2 * time.Second)

	// Send S1F1 (Are You There) and wait for response
	// This should timeout because equipment delays reply
	log.Println("Sending S1F1 (Are You There) with ExpectReply=True...")

	s1f1 := ast.NewDataMessage("AreYouThere", 1, 1, 1, "H->E", ast.NewListNode())

	_, err := host.SendAndWait(s1f1)
	if err != nil {
		log.Printf("SendAndWait returned error: %v (Expected T3 Timeout)", err)
		if err.Error() == "T3 timeout" {
			log.Println("✓ Correctly caught T3 timeout in application layer")
		}
	} else {
		log.Println("unexpected: received response despite delay")
	}

	// Wait a bit to ensure S9F9 is flushed to network
	time.Sleep(2 * time.Second)

	log.Println("Stopping Host...")
	host.Disable()
}
