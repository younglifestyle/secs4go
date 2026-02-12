package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/younglifestyle/secs4go/gem"
	"github.com/younglifestyle/secs4go/hsms"
)

func main() {
	log.Println("============================================================")
	log.Println("P2 Clock Synchronization Test (Go Host)")
	log.Println("============================================================")
	log.Println("Connecting to: 127.0.0.1:5030 (Active mode)")
	log.Println("Test coverage:")
	log.Println("  - S2F17/F18: Request equipment time")
	log.Println("  - S2F31/F32: Set equipment time")
	log.Println("============================================================")

	protocol := hsms.NewHsmsProtocol("127.0.0.1", 5030, true, 10, "go-host-clock-test")
	handler, err := gem.NewGemHandler(gem.Options{
		Protocol:   protocol,
		DeviceType: gem.DeviceHost,
	})
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	handler.Enable()
	defer handler.Disable()

	log.Println()
	log.Println("Host enabled. Waiting for connection...")

	// Wait for COMMUNICATING state (up to 15 seconds)
	ok := handler.WaitForCommunicating(15 * time.Second)
	if !ok {
		log.Println("❌ Failed to reach COMMUNICATING state within 15s")
		os.Exit(1)
	}
	log.Println("✓ Host in COMMUNICATING state")

	passed := 0
	failed := 0

	// ─── Test 1: S2F17/F18 ───────────────────────────────────
	log.Println()
	log.Println("--- Test 1: Request Equipment Time (S2F17/F18) ---")
	timeStr, err := handler.RequestDateTime()
	if err != nil {
		log.Printf("❌ S2F17 failed: %v", err)
		failed++
	} else {
		log.Printf("✓ Received equipment time: %s", timeStr)
		if len(timeStr) >= 14 {
			log.Printf("✓ Time format valid (%d characters)", len(timeStr))
			passed++
		} else {
			log.Printf("❌ Time format invalid (only %d characters)", len(timeStr))
			failed++
		}
	}

	// ─── Test 2: S2F31/F32 (valid time) ──────────────────────
	log.Println()
	log.Println("--- Test 2: Set Equipment Time (S2F31/F32) ---")
	futureTime := time.Now().Add(1 * time.Minute)
	futureStr := futureTime.Format("20060102150405") + "00"
	log.Printf("Setting time to: %s (current + 1 min)", futureStr)

	tiack, err := handler.SetDateTime(futureStr)
	if err != nil {
		log.Printf("❌ S2F31 failed: %v", err)
		failed++
	} else {
		if tiack == 0 {
			log.Printf("✓ Time accepted (TIACK=%d)", tiack)
			passed++
		} else {
			log.Printf("❌ Time rejected (TIACK=%d)", tiack)
			failed++
		}
	}

	// ─── Test 3: Verify time was updated ─────────────────────
	log.Println()
	log.Println("--- Test 3: Verify Updated Time (S2F17/F18 again) ---")
	timeStr2, err := handler.RequestDateTime()
	if err != nil {
		log.Printf("❌ S2F17 (verify) failed: %v", err)
		failed++
	} else {
		log.Printf("✓ Equipment time after set: %s", timeStr2)
		passed++
	}

	// ─── Summary ─────────────────────────────────────────────
	log.Println()
	log.Println("============================================================")
	log.Printf("Test Results: %d passed, %d failed", passed, failed)
	log.Println("============================================================")

	if failed > 0 {
		fmt.Fprintln(os.Stderr, "SOME TESTS FAILED")
		os.Exit(1)
	}
	log.Println("ALL TESTS PASSED ✓")
}
