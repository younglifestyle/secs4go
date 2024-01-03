package secs

//import (
//	"log"
//	"math/rand"
//	"sync"
//	"time"
//)
//
//// HsmsProtocol represents the base class for creating Host/Equipment models.
//type HsmsProtocol struct {
//	address             string
//	port                int
//	active              bool
//	sessionID           int
//	name                string
//	customConnection    HsmsConnectionHandler
//	logger              *log.Logger
//	communicationLogger *log.Logger
//	connected           bool
//	systemCounter       int
//	linktestTimer       *time.Timer
//	linktestTimeout     time.Duration
//	selectReqThread     *sync.WaitGroup
//	systemQueues        map[int]chan HsmsPacket
//	connectionState     *ConnectionStateMachine
//	connection          HsmsConnection
//}
//
//// NewHsmsProtocol creates a new HsmsProtocol instance.
//func NewHsmsProtocol(address string, port int, active bool, sessionID int, name string, customConnectionHandler HsmsConnectionHandler) *HsmsProtocol {
//	logger := log.New(log.Writer(), "hsms_protocol: ", log.LstdFlags)
//	communicationLogger := log.New(log.Writer(), "hsms_communication: ", log.LstdFlags)
//
//	h := &HsmsProtocol{
//		address:             address,
//		port:                port,
//		active:              active,
//		sessionID:           sessionID,
//		name:                name,
//		customConnection:    customConnectionHandler,
//		logger:              logger,
//		communicationLogger: communicationLogger,
//		connected:           false,
//		systemCounter:       rand.Int(),
//		linktestTimer:       nil,
//		linktestTimeout:     30 * time.Second,
//		selectReqThread:     nil,
//		systemQueues:        make(map[int]chan HsmsPacket),
//		connectionState:     NewConnectionStateMachine(),
//		connection:          nil,
//	}
//
//	h.connectionState.OnStateEnter(ConnectedState, h.onStateConnect)
//	h.connectionState.OnStateExit(ConnectedState, h.onStateDisconnect)
//	h.connectionState.OnStateEnter(ConnectedSelectedState, h.onStateSelect)
//
//	// Setup connection
//	if h.active {
//		if h.customConnection == nil {
//			h.connection = NewHsmsActiveConnection(h.address, h.port, h.sessionID, h)
//		} else {
//			h.connection = h.customConnection.CreateConnection(h.address, h.port, h.sessionID, h)
//		}
//	} else {
//		if h.customConnection == nil {
//			h.connection = NewHsmsPassiveConnection(h.address, h.port, h.sessionID, h)
//		} else {
//			h.connection = h.customConnection.CreateConnection(h.address, h.port, h.sessionID, h)
//		}
//	}
//
//	return h
//}
