package gem

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
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
	equipmentProtocol.Timeouts().SetT3ReplyTimeout(10)
	hostProtocol.Timeouts().SetLinktest(60)
	hostProtocol.Timeouts().SetT3ReplyTimeout(10)

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

	equipmentHandler.SetRemoteCommandHandler(func(req RemoteCommandRequest) (RemoteCommandResult, error) {
		switch req.Command {
		case "START":
			lot := "RUN"
			for _, param := range req.Parameters {
				if param.Name == "LOTID" {
					if ascii, ok := param.Value.(*ast.ASCIINode); ok {
						if text, ok := ascii.Values().(string); ok && text != "" {
							lot = text
						}
					}
					break
				}
			}
			state.startLot(lot)
			return RemoteCommandResult{HCACK: HCACKAcknowledge}, nil
		case "STOP":
			state.stopLot()
			return RemoteCommandResult{HCACK: HCACKAcknowledge}, nil
		default:
			return RemoteCommandResult{HCACK: HCACKInvalidCommand}, fmt.Errorf("unknown command %q", req.Command)
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
	if os.Getenv("SECS4GO_STRESS") == "" {
		t.Skip("set SECS4GO_STRESS=1 to run soak workload")
	}

	equipment, host, state, cleanup := startPairedHandlers(t)
	defer cleanup()

	const (
		goroutines  = 16
		iterations  = 32000
		soakTimeout = 30 * time.Second
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout := time.NewTimer(soakTimeout)
	defer timeout.Stop()

	errCh := make(chan error, goroutines+1)

	var statusRequests int64
	var remoteCommands int64
	var ceidTriggers int64

	start := time.Now()
	timedOut := false

	var requestWG sync.WaitGroup
	requestWG.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer requestWG.Done()
			for j := 0; j < iterations; j++ {
				if ctx.Err() != nil {
					return
				}

				if host.State() != CommunicationStateCommunicating {
					if !host.WaitForCommunicating(5 * time.Second) {
						errCh <- fmt.Errorf("goroutine %d host lost communication", idx)
						return
					}
				}
				if equipment.State() != CommunicationStateCommunicating {
					if !equipment.WaitForCommunicating(5 * time.Second) {
						errCh <- fmt.Errorf("goroutine %d equipment lost communication", idx)
						return
					}
				}

				if _, err := host.RequestStatusVariables(int64(1001)); err != nil {
					errCh <- fmt.Errorf("goroutine %d status request failed: %w", idx, err)
					return
				}
				atomic.AddInt64(&statusRequests, 1)

				cmd := "START"
				paramValues := []RemoteCommandParameterValue{{Name: "LOTID", Value: fmt.Sprintf("LOT-%d-%d", idx, j)}}
				if j%2 == 1 {
					cmd = "STOP"
					paramValues = nil
				}

				result, err := host.SendRemoteCommand(cmd, paramValues)
				if err != nil || result.HCACK != HCACKAcknowledge {
					errCh <- fmt.Errorf("goroutine %d remote command %s failed: hcack=%d err=%v", idx, cmd, result.HCACK, err)
					return
				}
				atomic.AddInt64(&remoteCommands, 1)

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
				atomic.AddInt64(&ceidTriggers, 1)
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

	// ---- 等待完成或超时；超时也打印 summary ----
	select {
	case <-done:
		// 正常完成，继续走下面的校验与 summary
	case <-timeout.C:
		timedOut = true
		cancel()

		// 立刻用当前计数和当前时间打印 summary
		duration := time.Since(start)
		sv := atomic.LoadInt64(&statusRequests)
		rc := atomic.LoadInt64(&remoteCommands)
		ce := atomic.LoadInt64(&ceidTriggers)
		totalOps := float64(sv + rc)
		throughput := totalOps / duration.Seconds()

		t.Logf("[TIMEOUT SUMMARY] goroutines=%d iterations=%d duration=%s status_ops=%d remote_ops=%d ce_triggers=%d throughput=%.1f ops/s",
			goroutines, iterations, duration, sv, rc, ce, throughput)

		t.Fatalf("soak test timed out after %s", soakTimeout)
	}

	// ---- 正常完成路径：严格校验并打印 summary ----
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	duration := time.Since(start)
	expectedOps := int64(goroutines * iterations)

	if got := atomic.LoadInt64(&statusRequests); got != expectedOps {
		t.Fatalf("status variable request count mismatch: got %d want %d", got, expectedOps)
	}
	if got := atomic.LoadInt64(&remoteCommands); got != expectedOps {
		t.Fatalf("remote command count mismatch: got %d want %d", got, expectedOps)
	}
	ceCount := atomic.LoadInt64(&ceidTriggers)
	if ceCount == 0 {
		t.Fatalf("collection event triggers not observed")
	}

	totalOps := float64(statusRequests + remoteCommands)
	throughput := totalOps / duration.Seconds()
	t.Logf("soak summary: goroutines=%d iterations=%d duration=%s status_ops=%d remote_ops=%d ce_triggers=%d throughput=%.1f ops/s",
		goroutines, iterations, duration, statusRequests, remoteCommands, ceCount, throughput)

	_ = timedOut // 仅为避免未使用警告，实际不需要
}

/*
status_ops=%d
已完成的 状态变量读取 操作次数（每次循环调用一次 RequestStatusVariables，等价于完成的 S1F3/S1F11 类请求数）。

remote_ops=%d
已完成的 远程命令 操作次数（每次循环调用一次 SendRemoteCommand，等价于完成的 S2F41 次数）。

ce_triggers=%d
已触发的 采集事件触发 次数（后台 goroutine 周期性调用 TriggerCollectionEvent，通常对应设备上报 S6F11 的频次）。

throughput=%.1f ops/s
吞吐率（每秒操作数），注意这里只统计主动发起的两类请求（状态+远程命令），不把 ce_triggers 算进吞吐率里。
*/
