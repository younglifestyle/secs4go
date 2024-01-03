package hsms

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/looplab/fsm"
	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/codec"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/parser/hsms"
	"github.com/younglifestyle/secs4go/utils"
	"go.uber.org/atomic"
	"log"
	"math/rand"
	"time"
)

// HsmsProtocol represents the base class for creating Host/Equipment models.
type HsmsProtocol struct {
	remoteAddress    string
	remotePort       int
	active           bool
	sessionID        int
	name             string
	enabled          bool
	logger           *log.Logger
	connected        bool
	stopServerThread bool
	systemCounter    *atomic.Uint32
	linktestTimer    *time.Ticker
	disconnectFlg    chan struct{}
	systemQueues     map[uint32]*utils.Deque
	timeouts         *SecsTimeout
	connectionState  *ConnectionStateMachine
	hsmsConnection   *HsmsConnection
	//connection       *link.Session
	server *link.Server
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
		logger:        logger,
		connected:     false,
		systemCounter: atomic.NewUint32(rand.Uint32()),
		disconnectFlg: make(chan struct{}),
		timeouts:      NewSecsTimeout(),
		systemQueues:  make(map[uint32]*utils.Deque),
	}

	h.connectionState = NewConnectionStateMachine(fsm.Callbacks{
		"leave_NOT-CONNECTED":      h.onStateConnect,
		"enter_NOT-CONNECTED":      h.onStateDisconnect,
		"enter_CONNECTED_SELECTED": h.onStateSelect,
	})
	h.hsmsConnection = NewHsmsConnection(active, address, port, sessionID, h)

	return h
}

func (p *HsmsProtocol) startLinktestTimer() {
	log.Println("Linktest timer expired. Performing linktest...")

	p.linktestTimer = time.NewTicker(time.Duration(p.timeouts.Linktest()) * time.Second)
	defer func() {
		p.linktestTimer.Stop()
		p.linktestTimer = nil
	}()

	p.hsmsConnection.sendLinktestReq()
	for {
		select {
		case <-p.disconnectFlg:
			// 状态发生改变，直接退出循环
			return
		case <-p.linktestTimer.C:
			// send link test
			p.hsmsConnection.sendLinktestReq()
		}
	}
}

// 未做其他事，可以后期支持传入回调函数
func (p *HsmsProtocol) onStateSelect(ctx context.Context, e *fsm.Event) {
	p.logger.Println("onStateSelect")

}

func (p *HsmsProtocol) onStateDisconnect(ctx context.Context, e *fsm.Event) {
	p.logger.Println("onStateDisconnect")

	if p.linktestTimer != nil {
		p.disconnectFlg <- struct{}{}
	}
}

// onLinktestTimer is the callback function to be executed when the linktest timer expires.
func (p *HsmsProtocol) onStateConnect(ctx context.Context, e *fsm.Event) {
	log.Println("Linktest timer expired. Performing linktest...")

	go func() {
		// start select process if connection is active
		if p.active {
			err := p.hsmsConnection.sendSelectReq()
			if err != nil {
				fmt.Println("onStateConnect send select req err : ", err)
				return
			}
		}

		// start linktest timer
		p.startLinktestTimer()
	}()
}

func (p *HsmsProtocol) getNextSystemId(num uint32) []byte {
	var id = make([]byte, 4)
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

func (p *HsmsProtocol) getQueueForSystem(systemId uint32) *utils.Deque {
	p.systemQueues[systemId] = utils.NewDeque()
	return p.systemQueues[systemId]
}

func (p *HsmsProtocol) removeQueue(systemId uint32) {
	delete(p.systemQueues, systemId)
}

func (p *HsmsProtocol) OnConnectionEstablished() {
	p.connected = true
	p.connectionState.Connect()
}

func (p *HsmsProtocol) enable() {
	if !p.enabled {
		p.enabled = true
		go p.startConnectThread()
	}
}

func (p *HsmsProtocol) disable() {
	if !p.enabled {
		p.enabled = false

		if p.active {
			p.disconnectFlg <- struct{}{}
			p.hsmsConnection.Close()
		} else { // passive
			p.stopServerThread = true
			p.server.Stop()
		}
	}
}

func (p *HsmsProtocol) startConnectThread() {
	for {
		if p.active {
			err := p.activeConnect()
			if err != nil {
				p.logger.Println("connect error : ", err)
				// idle t5
				<-time.After(time.Duration(p.timeouts.t5ConnSeparateTimeout) * time.Second)
				continue
			} else {
				return
			}
		} else {
			if !p.stopServerThread {
				p.passiveConnect()
			}
		}
	}
}

func (p *HsmsProtocol) passiveConnect() {
	server, err := link.Listen("tcp",
		fmt.Sprintf("%s:%d", p.remoteAddress, p.remotePort),
		codec.Bufio(codec.SECSII(), 1024*5, 1024*5),
		0, link.HandlerFunc(p.OnConnectionEstablishedAndStartReceiver))
	if err != nil {
		p.logger.Println("listen tcp error : ", err)
		return
	}
	defer server.Stop()

	//err = server.Serve()
	err = server.SingleServe()
	if err != nil {
		p.logger.Println("start tcp server error : ", err)
		return
	}
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

// 处理控制信息
func (p *HsmsProtocol) handleHsmsRequests(message ast.HSMSMessage) {
	p.logger.Println("get control message : ", message.Type())

	switch message.Type() {
	case hsms.SelectReqStr:
		if !p.connected {
			p.hsmsConnection.sendReject(message, 4)
		} else {
			p.hsmsConnection.sendSelectRsp(message, 0)

			// change state
			err := p.connectionState.Select()
			if err != nil {
				p.logger.Println("change state to select error : ", err)
			}
		}
	case hsms.SelectRspStr:
		// change state
		err := p.connectionState.Select()
		if err != nil {
			p.logger.Println("change state to select error : ", err)
		}

		systemId := binary.BigEndian.Uint32(message.SystemBytes())
		if deque, ok := p.systemQueues[systemId]; ok {
			// send packet to request sender
			deque.Put(message)
		}
	case hsms.DeselectReqStr:
		if !p.connected {
			p.hsmsConnection.sendReject(message, 4)
		} else {
			p.hsmsConnection.sendDeselectRsp(message, 0)

			// update connection state
			err := p.connectionState.Deselect()
			if err != nil {
				p.logger.Println("change state to deselect error : ", err)
			}
		}

	case hsms.DeselectRspStr:
		// change state
		err := p.connectionState.Deselect()
		if err != nil {
			p.logger.Println("change state to select error : ", err)
		}

		systemId := binary.BigEndian.Uint32(message.SystemBytes())
		if deque, ok := p.systemQueues[systemId]; ok {
			// send packet to request sender
			deque.Put(message)
		}

	case hsms.LinktestReqStr:
		// if we are disconnecting send reject else send response
		if !p.connected {
			p.hsmsConnection.sendSelectRsp(message, 4)
		} else {
			p.hsmsConnection.sendLinkTestRsp(message)
		}
	default:
		systemId := binary.BigEndian.Uint32(message.SystemBytes())
		if deque, ok := p.systemQueues[systemId]; ok {
			// send packet to request sender
			deque.Put(message)
		}
	}
}

func (p *HsmsProtocol) OnConnectionEstablishedAndStartReceiver(connection *link.Session) {
	// passive 单链接, 后期采用参数去控制
	if !p.active && p.hsmsConnection.Session() == nil {
		p.hsmsConnection.SetSession(connection)
	}
	//if !p.active {
	//	if p.hsmsConnection.Session() == nil {
	//		p.hsmsConnection.SetSession(connection)
	//	} else if !p.hsmsConnection.Session().IsClosed() {
	//		connection.Close()
	//		return
	//	}
	//}

	// change state
	p.OnConnectionEstablished()

	// StartReceiver
	for {
		rsp, err := connection.Receive()
		if err != nil {
			p.logger.Println("recv error : ", err)
			p.connected = false
			// TODO 判断错误为异常错误，则调用预设的状态改变函数，改变state状态, 将onStateDisconnect实现
			p.connectionState.Disconnect()
			return
		}

		message := rsp.(ast.HSMSMessage)
		if message.Type() != hsms.DataMessageStr {
			p.handleHsmsRequests(message)
		} else {
			if p.connectionState.CurrentState() != StateConnectedSelected {
				p.logger.Println("received message when not selected")

				// Send RejectReqHeader
				p.hsmsConnection.sendReject(message, 4)
				continue
			}

			systemId := binary.BigEndian.Uint32(message.SystemBytes())
			// someone is waiting for this message
			if deque, ok := p.systemQueues[systemId]; ok {
				// send packet to request sender
				deque.Put(message)
			} else {
				// TODO check handler corresponding to stream function

			}
		}
	}
}
