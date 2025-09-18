package hsms

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/looplab/fsm"
	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/codec"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/parser/hsms"
	"github.com/younglifestyle/secs4go/utils"
	"go.uber.org/atomic"
)

var (
	// ErrNotConnected is returned when HSMS operations are invoked before the
	// TCP session is established.
	ErrNotConnected = errors.New("hsms: connection not established")
	// ErrNotSelected is returned when operations require the HSMS connection to
	// be in SELECTED state but it is not.
	ErrNotSelected = errors.New("hsms: connection not selected")
	// ErrTimeout indicates the remote side did not answer within the configured timeout.
	ErrTimeout = errors.New("hsms: timeout waiting for response")
)

type DataMessageHandler func(*ast.DataMessage) (*ast.DataMessage, error)

func streamFunctionKey(stream, function int) string {
	return fmt.Sprintf("S%02dF%02d", stream, function)
}

// HsmsProtocol represents the base class for creating Host/Equipment models.
type HsmsProtocol struct {
	remoteAddress string
	remotePort    int
	active        bool
	sessionID     int
	name          string

	enabled   *atomic.Bool
	connected *atomic.Bool

	logger *log.Logger

	systemCounter *atomic.Uint32

	timeouts        *SecsTimeout
	connectionState *ConnectionStateMachine
	hsmsConnection  *HsmsConnection

	queueMu      sync.RWMutex
	systemQueues map[uint32]*utils.Deque

	handlerMu      sync.RWMutex
	handlers       map[string]DataMessageHandler
	defaultHandler DataMessageHandler

	linktestTimerMu sync.Mutex
	linktestTimer   *time.Ticker
	disconnectFlg   chan struct{}

	serverMu sync.Mutex
	server   *link.Server
}

// NewHsmsProtocol creates a new HsmsProtocol instance.
func NewHsmsProtocol(address string, port int, active bool, sessionID int, name string) *HsmsProtocol {
	logger := log.New(log.Writer(), "hsms_protocol: ", log.LstdFlags)

	h := &HsmsProtocol{
		remoteAddress: address,
		remotePort:    port,
		active:        active,
		sessionID:     sessionID,
		name:          name,
		enabled:       atomic.NewBool(false),
		connected:     atomic.NewBool(false),
		logger:        logger,
		systemCounter: atomic.NewUint32(rand.Uint32()),
		timeouts:      NewSecsTimeout(),
		systemQueues:  make(map[uint32]*utils.Deque),
		handlers:      make(map[string]DataMessageHandler),
		disconnectFlg: make(chan struct{}, 1),
	}

	h.connectionState = NewConnectionStateMachine(fsm.Callbacks{
		"leave_NOT-CONNECTED":      h.onStateConnect,
		"enter_NOT-CONNECTED":      h.onStateDisconnect,
		"enter_CONNECTED_SELECTED": h.onStateSelect,
	})
	h.hsmsConnection = NewHsmsConnection(active, address, port, sessionID, h)

	return h
}

func (p *HsmsProtocol) clearDisconnectFlag() {
	for {
		select {
		case <-p.disconnectFlg:
		default:
			return
		}
	}
}

func (p *HsmsProtocol) startLinktestTimer() {
	ticker := time.NewTicker(time.Duration(p.timeouts.Linktest()) * time.Second)

	p.linktestTimerMu.Lock()
	p.linktestTimer = ticker
	p.linktestTimerMu.Unlock()

	defer func() {
		ticker.Stop()
		p.linktestTimerMu.Lock()
		if p.linktestTimer == ticker {
			p.linktestTimer = nil
		}
		p.linktestTimerMu.Unlock()
	}()

	p.clearDisconnectFlag()
	p.hsmsConnection.sendLinktestReq()

	for {
		select {
		case <-ticker.C:
			p.hsmsConnection.sendLinktestReq()
		case <-p.disconnectFlg:
			return
		}
	}
}

func (p *HsmsProtocol) stopLinktestTimer() {
	p.linktestTimerMu.Lock()
	defer p.linktestTimerMu.Unlock()

	if p.linktestTimer != nil {
		select {
		case p.disconnectFlg <- struct{}{}:
		default:
		}
	}
}

// onStateSelect is triggered when the HSMS connection enters SELECTED state.
func (p *HsmsProtocol) onStateSelect(ctx context.Context, _ *fsm.Event) {
	p.logger.Println("state: CONNECTED_SELECTED")
}

func (p *HsmsProtocol) onStateDisconnect(ctx context.Context, _ *fsm.Event) {
	p.logger.Println("state: NOT_CONNECTED")
	p.connected.Store(false)
	p.stopLinktestTimer()
}

// onStateConnect is triggered when the HSMS connection transitions out of NOT-CONNECTED.
func (p *HsmsProtocol) onStateConnect(ctx context.Context, _ *fsm.Event) {
	p.logger.Println("state: CONNECTED_NOT_SELECTED")

	go func() {
		if p.active {
			if err := p.hsmsConnection.sendSelectReq(); err != nil {
				p.logger.Println("send select.req error:", err)
				return
			}
		}

		if !p.enabled.Load() {
			return
		}

		p.startLinktestTimer()
	}()
}

func (p *HsmsProtocol) encodeSystemID(num uint32) []byte {
	id := make([]byte, 4)
	binary.BigEndian.PutUint32(id, num)
	return id
}

func (p *HsmsProtocol) getNextSystemCounter() uint32 {
	num := int(p.systemCounter.Inc())
	if num > (2<<32)-1 {
		p.systemCounter.Store(0)
	}
	return uint32(num)
}

func (p *HsmsProtocol) createQueue(systemID uint32) *utils.Deque {
	queue := utils.NewDeque()
	p.queueMu.Lock()
	p.systemQueues[systemID] = queue
	p.queueMu.Unlock()
	return queue
}

func (p *HsmsProtocol) fetchQueue(systemID uint32) (*utils.Deque, bool) {
	p.queueMu.RLock()
	queue, ok := p.systemQueues[systemID]
	p.queueMu.RUnlock()
	return queue, ok
}

func (p *HsmsProtocol) removeQueue(systemID uint32) {
	p.queueMu.Lock()
	delete(p.systemQueues, systemID)
	p.queueMu.Unlock()
}

func (p *HsmsProtocol) OnConnectionEstablished() {
	p.connected.Store(true)
	if err := p.connectionState.Connect(); err != nil {
		p.logger.Println("change state to CONNECTED error:", err)
	}
}

// Enable starts the connection manager.
func (p *HsmsProtocol) Enable() {
	if p.enabled.CompareAndSwap(false, true) {
		go p.startConnectThread()
	}
}

// Disable stops the HSMS connection and releases resources.
func (p *HsmsProtocol) Disable() {
	if !p.enabled.CompareAndSwap(true, false) {
		return
	}

	p.stopLinktestTimer()
	p.connected.Store(false)

	if p.active {
		select {
		case p.disconnectFlg <- struct{}{}:
		default:
		}
		if err := p.hsmsConnection.Close(); err != nil {
			p.logger.Println("close session error:", err)
		}
	} else {
		p.serverMu.Lock()
		if p.server != nil {
			p.server.Stop()
			p.server = nil
		}
		p.serverMu.Unlock()
	}
}

func (p *HsmsProtocol) startConnectThread() {
	for p.enabled.Load() {
		if p.active {
			if err := p.activeConnect(); err != nil {
				if !p.enabled.Load() {
					return
				}
				p.logger.Println("connect error:", err)
				time.Sleep(time.Duration(p.timeouts.T5ConnSeparateTimeout()) * time.Second)
				continue
			}
			return
		}

		if err := p.passiveConnect(); err != nil {
			if !p.enabled.Load() {
				return
			}
			p.logger.Println("listen error:", err)
			time.Sleep(time.Second)
			continue
		}
	}
}

func (p *HsmsProtocol) passiveConnect() error {
	server, err := link.Listen("tcp",
		fmt.Sprintf("%s:%d", p.remoteAddress, p.remotePort),
		codec.Bufio(codec.SECSII(), 1024*5, 1024*5),
		0, link.HandlerFunc(p.OnConnectionEstablishedAndStartReceiver))
	if err != nil {
		return err
	}

	p.serverMu.Lock()
	p.server = server
	p.serverMu.Unlock()

	err = server.SingleServe()

	p.serverMu.Lock()
	if p.server == server {
		p.server = nil
	}
	p.serverMu.Unlock()

	server.Stop()

	if err != nil && p.enabled.Load() {
		return err
	}
	return nil
}

func (p *HsmsProtocol) activeConnect() error {
	session, err := link.Dial("tcp", fmt.Sprintf("%s:%d", p.remoteAddress, p.remotePort),
		codec.Bufio(codec.SECSII(), 1024*5, 1024*5), 0)
	if err != nil {
		return err
	}

	p.hsmsConnection.SetSession(session)
	p.OnConnectionEstablishedAndStartReceiver(session)

	return nil
}

// handleHsmsRequests processes HSMS control messages.
func (p *HsmsProtocol) handleHsmsRequests(message ast.HSMSMessage) {
	p.logger.Println("control message:", message.Type())

	systemID := binary.BigEndian.Uint32(message.SystemBytes())

	switch message.Type() {
	case hsms.SelectReqStr:
		ctrl, ok := message.(*ast.ControlMessage)
		if !ok {
			p.logger.Println("received malformed select.req")
			p.hsmsConnection.sendReject(message, 2)
			return
		}

		receivedSession := ctrl.SessionID()
		expectedSession := uint16(p.sessionID)
		fmt.Println(receivedSession, expectedSession)
		if receivedSession != expectedSession {
			// 0xFFFF0000 是 HSMS 标准规定的“系统消息会话 ID”，只用于 Select/Linktest 等控制消息。
			acceptMismatch := false
			if receivedSession == 0xFFFF {
				p.logger.Printf(
					"select.req session mismatch: remote used wildcard 0xFFFF, accepting expected %d",
					expectedSession,
				)
				acceptMismatch = true
			} else if expectedSession == 0 || expectedSession == 0xFFFF {
				p.logger.Printf(
					"select.req session mismatch: adopting remote session %d in place of expected %d",
					receivedSession,
					expectedSession,
				)
				p.sessionID = int(receivedSession)
				p.hsmsConnection.sessionID = int(receivedSession)
				acceptMismatch = true
			}
			if !acceptMismatch {
				p.logger.Printf("select.req session mismatch: got %d expected %d", receivedSession, expectedSession)
				p.hsmsConnection.sendSelectRsp(message, 1)
				_ = p.hsmsConnection.Close()
				p.connected.Store(false)
				if err := p.connectionState.Disconnect(); err != nil {
					p.logger.Println("change state to NOT-CONNECTED error:", err)
				}
				return
			}
		}

		if !p.connected.Load() {
			p.hsmsConnection.sendReject(message, 4)
			return
		}
		p.hsmsConnection.sendSelectRsp(message, 0)
		if err := p.connectionState.Select(); err != nil {
			p.logger.Println("change state to SELECTED error:", err)
		}

	case hsms.SelectRspStr:
		if queue, ok := p.fetchQueue(systemID); ok {
			queue.Put(message)
		}
		ctrl, ok := message.(*ast.ControlMessage)
		if !ok {
			p.logger.Println("received malformed select.rsp")
			return
		}
		if ctrl.Status() != 0 {
			p.logger.Printf("select.rsp returned status %d", ctrl.Status())
			p.connected.Store(false)
			if err := p.connectionState.Disconnect(); err != nil {
				p.logger.Println("change state to NOT-CONNECTED error:", err)
			}
			_ = p.hsmsConnection.Close()
			return
		}
		if err := p.connectionState.Select(); err != nil {
			p.logger.Println("change state to SELECTED error:", err)
		}

	case hsms.DeselectReqStr:
		if !p.connected.Load() {
			p.hsmsConnection.sendReject(message, 4)
			return
		}
		p.hsmsConnection.sendDeselectRsp(message, 0)
		if err := p.connectionState.Deselect(); err != nil {
			p.logger.Println("change state to NOT-SELECTED error:", err)
		}

	case hsms.DeselectRspStr:
		if err := p.connectionState.Deselect(); err != nil {
			p.logger.Println("change state to NOT-SELECTED error:", err)
		}
		if queue, ok := p.fetchQueue(systemID); ok {
			queue.Put(message)
		}

	case hsms.LinktestReqStr:
		if !p.connected.Load() {
			p.hsmsConnection.sendReject(message, 4)
			return
		}
		p.hsmsConnection.sendLinkTestRsp(message)

	default:
		if queue, ok := p.fetchQueue(systemID); ok {
			queue.Put(message)
		}
	}
}

func (p *HsmsProtocol) lookupHandler(stream, function int) DataMessageHandler {
	p.handlerMu.RLock()
	handler := p.handlers[streamFunctionKey(stream, function)]
	p.handlerMu.RUnlock()
	return handler
}

func (p *HsmsProtocol) invokeHandler(handler DataMessageHandler, msg *ast.DataMessage) {
	response, err := handler(msg)
	if err != nil {
		p.logger.Printf("handler error for S%02dF%02d: %v", msg.StreamCode(), msg.FunctionCode(), err)
		return
	}

	if response == nil {
		return
	}

	response = response.SetSessionIDAndSystemBytes(p.sessionID, msg.SystemBytes())
	if response.WaitBit() == "optional" {
		response = response.SetWaitBit(false)
	}

	if err := p.hsmsConnection.Send(response.ToBytes()); err != nil {
		p.logger.Printf("send response S%02dF%02d failed: %v", response.StreamCode(), response.FunctionCode(), err)
	}
}

func (p *HsmsProtocol) handleDataMessage(message *ast.DataMessage) {
	handler := p.lookupHandler(message.StreamCode(), message.FunctionCode())
	if handler != nil {
		p.invokeHandler(handler, message)
		return
	}

	p.handlerMu.RLock()
	defaultHandler := p.defaultHandler
	p.handlerMu.RUnlock()

	if defaultHandler != nil {
		p.invokeHandler(defaultHandler, message)
		return
	}

	p.logger.Printf("unhandled data message S%02dF%02d", message.StreamCode(), message.FunctionCode())
}

func (p *HsmsProtocol) OnConnectionEstablishedAndStartReceiver(connection *link.Session) {
	if !p.active {
		if p.hsmsConnection.Session() == nil {
			p.hsmsConnection.SetSession(connection)
		} else if !p.hsmsConnection.Session().IsClosed() {
			_ = connection.Close()
			return
		} else {
			p.hsmsConnection.SetSession(connection)
		}
	}

	p.OnConnectionEstablished()

	for p.enabled.Load() {
		rsp, err := connection.Receive()
		if err != nil {
			p.logger.Println("receive error:", err)
			p.connected.Store(false)
			p.stopLinktestTimer()
			if err := p.connectionState.Disconnect(); err != nil {
				p.logger.Println("change state to NOT-CONNECTED error:", err)
			}
			return
		}

		message := rsp.(ast.HSMSMessage)
		if message.Type() != hsms.DataMessageStr {
			p.handleHsmsRequests(message)
			continue
		}

		dataMessage, ok := message.(*ast.DataMessage)
		if !ok {
			p.logger.Printf("unexpected HSMS message type %T", message)
			continue
		}

		if p.connectionState.CurrentState() != StateConnectedSelected {
			p.logger.Println("received message while not selected")
			p.hsmsConnection.sendReject(message, 4)
			continue
		}

		systemID := binary.BigEndian.Uint32(dataMessage.SystemBytes())
		if queue, ok := p.fetchQueue(systemID); ok {
			queue.Put(dataMessage)
			continue
		}

		p.handleDataMessage(dataMessage)
	}
}

func (p *HsmsProtocol) ensureReady(requireSelected bool) error {
	if !p.connected.Load() || p.hsmsConnection.Session() == nil {
		return ErrNotConnected
	}
	if requireSelected && p.connectionState.CurrentState() != StateConnectedSelected {
		return ErrNotSelected
	}
	return nil
}

// SendDataMessage sends a SECS-II data message without waiting for the reply.
func (p *HsmsProtocol) SendDataMessage(message *ast.DataMessage) error {
	if message == nil {
		return errors.New("hsms: nil message")
	}
	if err := p.ensureReady(true); err != nil {
		return err
	}

	systemID := p.getNextSystemCounter()
	outgoing := message.SetSessionIDAndSystemBytes(p.sessionID, p.encodeSystemID(systemID))
	if outgoing.WaitBit() == "optional" {
		outgoing = outgoing.SetWaitBit(false)
	}

	return p.hsmsConnection.Send(outgoing.ToBytes())
}

// SendAndWait sends a SECS-II data message and waits for the response.
func (p *HsmsProtocol) SendAndWait(message *ast.DataMessage) (*ast.DataMessage, error) {
	if message == nil {
		return nil, errors.New("hsms: nil message")
	}
	if err := p.ensureReady(true); err != nil {
		return nil, err
	}

	systemID := p.getNextSystemCounter()
	queue := p.createQueue(systemID)
	defer p.removeQueue(systemID)

	outgoing := message.SetSessionIDAndSystemBytes(p.sessionID, p.encodeSystemID(systemID))
	if outgoing.WaitBit() == "optional" {
		outgoing = outgoing.SetWaitBit(true)
	}

	if err := p.hsmsConnection.Send(outgoing.ToBytes()); err != nil {
		return nil, err
	}

	resp, err := queue.Get(p.timeouts.T3ReplyTimeout())
	if err != nil {
		return nil, ErrTimeout
	}

	switch value := resp.(type) {
	case *ast.DataMessage:
		return value, nil
	case ast.HSMSMessage:
		dataMessage, ok := value.(*ast.DataMessage)
		if !ok {
			return nil, fmt.Errorf("hsms: unexpected response type %T", value)
		}
		return dataMessage, nil
	default:
		return nil, fmt.Errorf("hsms: unexpected response payload %T", resp)
	}
}

// SendResponse transmits a SECS-II reply reusing the system bytes from the request.
func (p *HsmsProtocol) SendResponse(message *ast.DataMessage, systemBytes []byte) error {
	if message == nil {
		return errors.New("hsms: nil message")
	}
	if err := p.ensureReady(true); err != nil {
		return err
	}

	outgoing := message.SetSessionIDAndSystemBytes(p.sessionID, systemBytes)
	if outgoing.WaitBit() == "optional" {
		outgoing = outgoing.SetWaitBit(false)
	}

	return p.hsmsConnection.Send(outgoing.ToBytes())
}

// RegisterHandler registers a callback for a specific stream/function pair.
func (p *HsmsProtocol) RegisterHandler(stream, function int, handler DataMessageHandler) {
	key := streamFunctionKey(stream, function)
	p.handlerMu.Lock()
	defer p.handlerMu.Unlock()

	if handler == nil {
		delete(p.handlers, key)
		return
	}

	p.handlers[key] = handler
}

// RegisterDefaultHandler registers a fallback handler invoked when no specific handler exists.
func (p *HsmsProtocol) RegisterDefaultHandler(handler DataMessageHandler) {
	p.handlerMu.Lock()
	p.defaultHandler = handler
	p.handlerMu.Unlock()
}

// Connected reports whether the underlying HSMS session is established.
func (p *HsmsProtocol) Connected() bool {
	return p.connected.Load()
}

// CurrentState returns the current connection state name.
func (p *HsmsProtocol) CurrentState() string {
	return p.connectionState.CurrentState()
}

// Timeouts returns protocol timeout configuration.
func (p *HsmsProtocol) Timeouts() *SecsTimeout {
	return p.timeouts
}
