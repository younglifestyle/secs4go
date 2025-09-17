package gem

import (
	"errors"
	"log"
	"sync"
	"time"

	"go.uber.org/atomic"

	"github.com/younglifestyle/secs4go/common"
	"github.com/younglifestyle/secs4go/hsms"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

var (
	// ErrNotCommunicating is returned when an operation requires GEM communicating state.
	ErrNotCommunicating = errors.New("gem: not in communicating state")
	// ErrOperationNotSupported indicates the requested operation is invalid for the current device type.
	ErrOperationNotSupported = errors.New("gem: operation not supported for this device type")
)

// Events exposes GEM handler callbacks.
type Events struct {
	HandlerCommunicating  *common.Event
	AlarmReceived         *common.Event
	AlarmAckReceived      *common.Event
	RemoteCommandReceived *common.Event
}

// GemHandler orchestrates GEM handshake and selected services on top of HSMS protocol.
type GemHandler struct {
	protocol   *hsms.HsmsProtocol
	deviceType DeviceType
	deviceID   uint16
	mdln       string
	softrev    string

	establishWait time.Duration

	state   *stateMachine
	control *ControlStateMachine

	enabled             *atomic.Bool
	handshakeInProgress *atomic.Bool

	events Events

	waitersMu sync.Mutex
	waiters   []chan struct{}

	stopMu sync.Mutex
	stopCh chan struct{}

	alarmMu sync.RWMutex
	alarms  map[int]Alarm

	remoteMu             sync.RWMutex
	remoteCommandHandler RemoteCommandHandler

	logger *log.Logger
}

// NewGemHandler creates a GEM handler backed by the provided HSMS protocol.
func NewGemHandler(opts Options) (*GemHandler, error) {
	if opts.Protocol == nil {
		return nil, errors.New("gem: protocol is required")
	}
	opts.applyDefaults()

	handler := &GemHandler{
		protocol:      opts.Protocol,
		deviceType:    opts.DeviceType,
		deviceID:      opts.DeviceID,
		mdln:          opts.MDLN,
		softrev:       opts.SOFTREV,
		establishWait: opts.EstablishCommunicationWait,
		state:         newStateMachine(),
		control: NewControlStateMachine(ControlStateMachineOptions{
			InitialState:      opts.InitialControlState,
			InitialOnlineMode: opts.InitialOnlineMode,
		}),
		enabled:             atomic.NewBool(false),
		handshakeInProgress: atomic.NewBool(false),
		events: Events{
			HandlerCommunicating:  &common.Event{},
			AlarmReceived:         &common.Event{},
			AlarmAckReceived:      &common.Event{},
			RemoteCommandReceived: &common.Event{},
		},
		alarms: make(map[int]Alarm),
		logger: log.New(log.Writer(), "gem_handler: ", log.LstdFlags),
	}

	handler.state.setState(CommunicationStateNotCommunicating)

	handler.protocol.RegisterHandler(1, 1, handler.onS1F1)
	handler.protocol.RegisterHandler(1, 13, handler.onS1F13)
	handler.protocol.RegisterHandler(1, 14, handler.onS1F14)

	if handler.deviceType == DeviceHost {
		handler.protocol.RegisterHandler(5, 1, handler.onS5F1)
	} else {
		handler.protocol.RegisterHandler(2, 41, handler.onS2F41)
	}

	handler.protocol.RegisterHandler(5, 2, handler.onS5F2)

	return handler, nil
}

// Events returns GEM handler event hooks.
func (g *GemHandler) Events() Events {
	return g.events
}

// Enable activates the underlying HSMS protocol and starts communication monitoring.
func (g *GemHandler) Enable() {
	if !g.enabled.CompareAndSwap(false, true) {
		return
	}

	g.state.setState(CommunicationStateNotCommunicating)
	g.protocol.Enable()

	g.stopMu.Lock()
	stopCh := make(chan struct{})
	g.stopCh = stopCh
	g.stopMu.Unlock()

	go g.monitorLoop(stopCh)
}

// Disable stops monitoring and disables the HSMS protocol.
func (g *GemHandler) Disable() {
	if !g.enabled.CompareAndSwap(true, false) {
		return
	}

	g.stopMu.Lock()
	if g.stopCh != nil {
		close(g.stopCh)
		g.stopCh = nil
	}
	g.stopMu.Unlock()

	g.state.stopTimers()
	g.protocol.Disable()
}

// State returns current communication state.
func (g *GemHandler) State() CommunicationState {
	return g.state.State()
}

// ControlState returns current control state.
func (g *GemHandler) ControlState() ControlState {
	if g.control == nil {
		return ControlStateInit
	}
	return g.control.State()
}

// ControlStateMachine exposes the underlying control state machine for advanced workflows.
func (g *GemHandler) ControlStateMachine() *ControlStateMachine {
	return g.control
}

// WaitForCommunicating blocks until the handler reaches communicating state or timeout is hit.
func (g *GemHandler) WaitForCommunicating(timeout time.Duration) bool {
	if g.State() == CommunicationStateCommunicating {
		return true
	}

	waiter := make(chan struct{}, 1)

	g.waitersMu.Lock()
	if g.State() == CommunicationStateCommunicating {
		g.waitersMu.Unlock()
		return true
	}
	g.waiters = append(g.waiters, waiter)
	g.waitersMu.Unlock()

	if timeout <= 0 {
		<-waiter
		return true
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-waiter:
		return true
	case <-timer.C:
		g.removeWaiter(waiter)
		return false
	}
}

func (g *GemHandler) removeWaiter(target chan struct{}) {
	g.waitersMu.Lock()
	defer g.waitersMu.Unlock()

	for idx, waiter := range g.waiters {
		if waiter == target {
			g.waiters = append(g.waiters[:idx], g.waiters[idx+1:]...)
			return
		}
	}
}

func (g *GemHandler) notifyCommunicating() {
	g.waitersMu.Lock()
	waiters := g.waiters
	g.waiters = nil
	g.waitersMu.Unlock()

	for _, waiter := range waiters {
		select {
		case waiter <- struct{}{}:
		default:
		}
	}

	if g.events.HandlerCommunicating != nil {
		g.events.HandlerCommunicating.Fire(map[string]interface{}{"handler": g})
	}
}

func (g *GemHandler) monitorLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			g.observe()
		}
	}
}

func (g *GemHandler) observe() {
	if !g.enabled.Load() {
		return
	}

	if !g.protocol.Connected() {
		g.handshakeInProgress.Store(false)
		g.state.setState(CommunicationStateNotCommunicating)
		return
	}

	if g.deviceType == DeviceHost && g.protocol.CurrentState() == hsms.StateConnectedSelected {
		if g.State() == CommunicationStateCommunicating {
			return
		}
		if g.handshakeInProgress.Load() {
			return
		}
		g.initiateHandshake()
	}
}

func (g *GemHandler) ensureCommunicating() error {
	if g.State() != CommunicationStateCommunicating {
		return ErrNotCommunicating
	}
	return nil
}

func (g *GemHandler) initiateHandshake() {
	if !g.enabled.Load() {
		return
	}

	if !g.handshakeInProgress.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer g.handshakeInProgress.Store(false)

		waitCRA := time.Duration(g.protocol.Timeouts().T3ReplyTimeout()) * time.Second
		if waitCRA <= 0 {
			waitCRA = 45 * time.Second
		}

		g.state.setStateWithWaitCRA(CommunicationStateWaitCRA, waitCRA, g.onWaitCRATimeout)

		dataMsg, err := g.protocol.SendAndWait(g.buildS1F13())
		if err != nil {
			g.logger.Println("establish communication request failed:", err)
			g.scheduleRetry()
			return
		}

		if commack := readCommAck(dataMsg); commack != 0 {
			g.logger.Printf("COMMACK=%d, establishing communication denied", commack)
			g.scheduleRetry()
			return
		}

		g.setCommunicating()
	}()
}

func (g *GemHandler) scheduleRetry() {
	if !g.enabled.Load() {
		return
	}
	g.state.setStateWithWaitDelay(CommunicationStateWaitDelay, g.establishWait, g.onWaitDelayTimeout)
}

func (g *GemHandler) setCommunicating() {
	prev := g.state.setState(CommunicationStateCommunicating)
	if prev != CommunicationStateCommunicating {
		g.notifyCommunicating()
	}
}

func (g *GemHandler) onWaitCRATimeout() {
	if !g.enabled.Load() {
		return
	}
	if g.State() != CommunicationStateWaitCRA {
		return
	}
	g.logger.Println("wait CRA timed out")
	g.scheduleRetry()
}

func (g *GemHandler) onWaitDelayTimeout() {
	if !g.enabled.Load() {
		return
	}
	if g.State() != CommunicationStateWaitDelay {
		return
	}

	if g.deviceType == DeviceHost && g.protocol.Connected() && g.protocol.CurrentState() == hsms.StateConnectedSelected {
		g.initiateHandshake()
		return
	}

	g.state.setState(CommunicationStateNotCommunicating)
}

func (g *GemHandler) onS1F1(_ *ast.DataMessage) (*ast.DataMessage, error) {
	if g.deviceType == DeviceHost {
		return g.buildS1F2Host(), nil
	}
	return g.buildS1F2Equipment(), nil
}

func (g *GemHandler) onS1F13(_ *ast.DataMessage) (*ast.DataMessage, error) {
	g.setCommunicating()

	if g.deviceType == DeviceHost {
		return g.buildS1F14Host(0), nil
	}
	return g.buildS1F14Equipment(0), nil
}

func (g *GemHandler) onS1F14(msg *ast.DataMessage) (*ast.DataMessage, error) {
	if msg == nil {
		return nil, nil
	}
	if commack := readCommAck(msg); commack == 0 {
		g.setCommunicating()
	} else {
		g.logger.Printf("received S1F14 with COMMACK=%d", commack)
		g.scheduleRetry()
	}
	return nil, nil
}

func (g *GemHandler) onS5F1(msg *ast.DataMessage) (*ast.DataMessage, error) {
	if msg == nil {
		return g.buildS5F2(1), nil
	}

	if event, err := parseAlarmMessage(msg); err != nil {
		g.logger.Println("failed to parse S5F1:", err)
	} else if g.events.AlarmReceived != nil {
		g.events.AlarmReceived.Fire(map[string]interface{}{"alarm": event})
	}

	return g.buildS5F2(0), nil
}

func (g *GemHandler) onS5F2(msg *ast.DataMessage) (*ast.DataMessage, error) {
	if msg == nil {
		return nil, nil
	}

	ack, err := readHCACK(msg)
	if err != nil {
		g.logger.Println("failed to parse S5F2:", err)
		return nil, err
	}

	if ack != 0 {
		g.logger.Printf("alarm acknowledge returned code %d", ack)
	}

	if g.events.AlarmAckReceived != nil {
		g.events.AlarmAckReceived.Fire(map[string]interface{}{"ack": ack})
	}

	return nil, nil
}

func (g *GemHandler) onS2F41(msg *ast.DataMessage) (*ast.DataMessage, error) {
	if msg == nil {
		return g.buildS2F42(5), nil
	}

	req, err := parseRemoteCommand(msg)
	if err != nil {
		g.logger.Println("failed to parse S2F41:", err)
		return g.buildS2F42(1), nil
	}

	if g.events.RemoteCommandReceived != nil {
		g.events.RemoteCommandReceived.Fire(map[string]interface{}{"request": req})
	}

	handler := g.getRemoteCommandHandler()
	if handler == nil {
		return g.buildS2F42(1), nil
	}

	ack, callErr := handler(req)
	if callErr != nil {
		g.logger.Println("remote command handler error:", callErr)
		if ack == 0 {
			ack = 1
		}
	}
	if ack < 0 || ack > 255 {
		ack = 1
	}
	return g.buildS2F42(ack), nil
}

func (g *GemHandler) buildS1F13() *ast.DataMessage {
	return ast.NewDataMessage("EstablishCommRequest", 1, 13, 1, "H->E", ast.NewListNode())
}

func (g *GemHandler) buildS1F14Host(commack int) *ast.DataMessage {
	body := ast.NewListNode(
		ast.NewBinaryNode(commack),
		ast.NewListNode(),
	)
	return ast.NewDataMessage("EstablishCommAck", 1, 14, 0, "H->E", body)
}

func (g *GemHandler) buildS1F14Equipment(commack int) *ast.DataMessage {
	body := ast.NewListNode(
		ast.NewBinaryNode(commack),
		ast.NewListNode(),
	)
	return ast.NewDataMessage("EstablishCommAck", 1, 14, 0, "H<-E", body)
}

func (g *GemHandler) buildS1F2Host() *ast.DataMessage {
	body := ast.NewListNode(ast.NewListNode())
	return ast.NewDataMessage("AreYouThereAck", 1, 2, 0, "H->E", body)
}

func (g *GemHandler) buildS1F2Equipment() *ast.DataMessage {
	md := ast.NewASCIINode(g.mdln)
	sr := ast.NewASCIINode(g.softrev)
	body := ast.NewListNode(ast.NewListNode(md, sr))
	return ast.NewDataMessage("AreYouThereAck", 1, 2, 0, "H<-E", body)
}

func readCommAck(msg *ast.DataMessage) int {
	if msg == nil {
		return -1
	}
	item, err := msg.Get(0)
	if err != nil {
		return -1
	}
	binaryNode, ok := item.(*ast.BinaryNode)
	if !ok {
		return -1
	}
	values, ok := binaryNode.Values().([]int)
	if !ok || len(values) == 0 {
		return -1
	}
	return values[0]
}

// SendRemoteCommand issues an S2F41 command (host only).
func (g *GemHandler) SendRemoteCommand(command string, params []string) (int, error) {
	if g.deviceType != DeviceHost {
		return -1, ErrOperationNotSupported
	}
	if err := g.ensureCommunicating(); err != nil {
		return -1, err
	}

	resp, err := g.protocol.SendAndWait(g.buildS2F41(command, params))
	if err != nil {
		return -1, err
	}
	if resp == nil {
		return -1, errors.New("gem: missing S2F42 response")
	}

	ack, err := readHCACK(resp)
	if err != nil {
		return -1, err
	}
	return ack, nil
}

// SetRemoteCommandHandler installs an equipment-side handler for S2F41 requests.
func (g *GemHandler) SetRemoteCommandHandler(handler RemoteCommandHandler) {
	g.remoteMu.Lock()
	defer g.remoteMu.Unlock()
	g.remoteCommandHandler = handler
}

func (g *GemHandler) getRemoteCommandHandler() RemoteCommandHandler {
	g.remoteMu.RLock()
	defer g.remoteMu.RUnlock()
	return g.remoteCommandHandler
}
