package gem

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

type equipmentState struct {
	mu          sync.Mutex
	lotID       string
	running     bool
	temperature int
}

func newEquipmentState() *equipmentState {
	return &equipmentState{lotID: "IDLE", temperature: 240}
}

func (s *equipmentState) startLot(lot string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lotID = lot
	s.running = true
}

func (s *equipmentState) stopLot() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lotID = "IDLE"
	s.running = false
}

func (s *equipmentState) lot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lotID
}

func (s *equipmentState) snapshotTemperature() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		if s.temperature < 280 {
			s.temperature++
		}
	} else if s.temperature > 200 {
		s.temperature--
	}
	return s.temperature
}

func (s *equipmentState) currentTemperature() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.temperature
}

func startPairedHandlers(t *testing.T) (*GemHandler, *GemHandler, *equipmentState, func()) {
	t.Helper()

	rand.Seed(time.Now().UnixNano())
	port := 40000 + rand.Intn(10000)

	equipmentProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, false, 0x100, "stress-eqp")
	hostProtocol := hsms.NewHsmsProtocol("127.0.0.1", port, true, 0x100, "stress-host")

	equipmentProtocol.Timeouts().SetLinktest(60)
	hostProtocol.Timeouts().SetLinktest(60)

	equipmentHandler, err := NewGemHandler(Options{
		Protocol:   equipmentProtocol,
		DeviceType: DeviceEquipment,
	})
	if err != nil {
		t.Fatalf("create equipment handler: %v", err)
	}

	hostHandler, err := NewGemHandler(Options{
		Protocol:   hostProtocol,
		DeviceType: DeviceHost,
	})
	if err != nil {
		t.Fatalf("create host handler: %v", err)
	}

	state := newEquipmentState()

	tempStatus, err := NewStatusVariable(1001, "Temperature", "C",
		WithStatusValueProvider(func() (ast.ItemNode, error) {
			return ast.NewUintNode(2, state.currentTemperature()), nil
		}),
	)
	if err != nil {
		t.Fatalf("create status variable: %v", err)
	}
	if err := equipmentHandler.RegisterStatusVariable(tempStatus); err != nil {
		t.Fatalf("register status variable: %v", err)
	}

	lotVar, err := NewDataVariable(2001, "LotID",
		WithDataValueProvider(func() (ast.ItemNode, error) {
			return ast.NewASCIINode(state.lot()), nil
		}),
	)
	if err != nil {
		t.Fatalf("create data variable: %v", err)
	}
	if err := equipmentHandler.RegisterDataVariable(lotVar); err != nil {
		t.Fatalf("register data variable: %v", err)
	}

	event, err := NewCollectionEvent(3001, "StateSnapshot")
	if err != nil {
		t.Fatalf("create collection event: %v", err)
	}
	if err := equipmentHandler.RegisterCollectionEvent(event); err != nil {
		t.Fatalf("register collection event: %v", err)
	}

	equipmentHandler.SetRemoteCommandHandler(func(req RemoteCommandRequest) (int, error) {
		switch req.Command {
		case "START":
			lot := "RUN"
			if len(req.Parameters) > 0 {
				lot = req.Parameters[0]
			}
			state.startLot(lot)
			return 0, nil
		case "STOP":
			state.stopLot()
			return 0, nil
		default:
			return 1, fmt.Errorf("unknown command %q", req.Command)
		}
	})

	equipmentHandler.Enable()
	hostHandler.Enable()

	const communicateTimeout = 15 * time.Second

	if !equipmentHandler.WaitForCommunicating(communicateTimeout) {
		t.Fatal("equipment failed to reach communicating state")
	}
	if !hostHandler.WaitForCommunicating(communicateTimeout) {
		t.Fatal("host failed to reach communicating state")
	}

	if ack, err := hostHandler.DefineReports(ReportDefinitionRequest{
		ReportID: 4001,
		VIDs:     []interface{}{int64(1001), int64(2001)},
	}); err != nil || ack != 0 {
		t.Fatalf("DefineReports failed ack=%d err=%v", ack, err)
	}
	if ack, err := hostHandler.LinkEventReports(EventReportLinkRequest{
		CEID:      int64(3001),
		ReportIDs: []interface{}{int64(4001)},
	}); err != nil || ack != 0 {
		t.Fatalf("LinkEventReports failed ack=%d err=%v", ack, err)
	}
	if ack, err := hostHandler.EnableEventReports(true, int64(3001)); err != nil || ack != 0 {
		t.Fatalf("EnableEventReports failed ack=%d err=%v", ack, err)
	}

	cleanup := func() {
		hostHandler.Disable()
		equipmentHandler.Disable()
	}

	return equipmentHandler, hostHandler, state, cleanup
}

func TestGemHighThroughputSoak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	equipment, host, state, cleanup := startPairedHandlers(t)
	defer cleanup()

	const (
		goroutines  = 10
		iterations  = 20
		soakTimeout = 20 * time.Second
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout := time.NewTimer(soakTimeout)
	defer timeout.Stop()

	errCh := make(chan error, goroutines+1)

	var requestWG sync.WaitGroup
	requestWG.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer requestWG.Done()
			for j := 0; j < iterations; j++ {
				if ctx.Err() != nil {
					return
				}

				if _, err := host.RequestStatusVariables(int64(1001)); err != nil {
					errCh <- fmt.Errorf("goroutine %d status request failed: %w", idx, err)
					return
				}

				cmd := "START"
				params := []string{fmt.Sprintf("LOT-%d-%d", idx, j)}
				if j%2 == 1 {
					cmd = "STOP"
					params = nil
				}

				if ack, err := host.SendRemoteCommand(cmd, params); err != nil || ack != 0 {
					errCh <- fmt.Errorf("goroutine %d remote command %s failed: ack=%d err=%v", idx, cmd, ack, err)
					return
				}

				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	var bgWG sync.WaitGroup
	bgWG.Add(1)
	go func() {
		defer bgWG.Done()
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = state.snapshotTemperature()
				if err := equipment.TriggerCollectionEvent(int64(3001)); err != nil {
					errCh <- fmt.Errorf("trigger collection event failed: %w", err)
					return
				}
			}
		}
	}()

	go func() {
		requestWG.Wait()
		cancel()
	}()

	done := make(chan struct{})
	go func() {
		bgWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-timeout.C:
		cancel()
		t.Fatalf("soak test timed out after %s", soakTimeout)
	}

	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}
