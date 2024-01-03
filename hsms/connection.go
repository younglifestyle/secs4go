package hsms

import (
	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
	"github.com/younglifestyle/secs4go/utils"
)

var HSMSSTYPES = map[int]string{
	1: "Select.req",
	2: "Select.rsp",
	3: "Deselect.req",
	4: "Deselect.rsp",
	5: "Linktest.req",
	6: "Linktest.rsp",
	7: "Reject.req",
	9: "Separate.req",
}

// 负责active与passive共同项的抽象
type HsmsConnection struct {
	// active: Is the connection active (*True*) or passive (*False*)
	active        bool
	remoteAddress string
	remotePort    int
	sessionID     int
	hp            *HsmsProtocol
	connection    *link.Session // 传入
}

func NewHsmsConnection(active bool, address string, port int, sessionID int, delegate *HsmsProtocol) *HsmsConnection {
	return &HsmsConnection{
		active:        active,
		remoteAddress: address,
		remotePort:    port,
		sessionID:     sessionID,
		hp:            delegate,
		connection:    nil,
	}
}

func (c *HsmsConnection) Session() *link.Session {
	return c.connection
}

func (c *HsmsConnection) SetSession(session *link.Session) {
	c.connection = session
}

func (c *HsmsConnection) Send(msg interface{}) error {
	return c.connection.Send(msg)
}

func (c *HsmsConnection) Close() error {
	return c.connection.Close()
}

func (c *HsmsConnection) sendRejectRsp(packet ast.HSMSMessage, reasonCode byte) {
	rejectReq := ast.NewHSMSMessageRejectReqFromMsg(packet, reasonCode)

	err := c.connection.Send(rejectReq)
	if err != nil {
		c.hp.logger.Println("send reject rsp error : ", err)
	}
}

func (c *HsmsConnection) sendLinktestReq() {
	systemId := c.hp.getNextSystemCounter()
	c.hp.systemQueues[systemId] = utils.NewDeque()
	defer delete(c.hp.systemQueues, systemId)

	err := c.connection.Send(ast.NewHSMSMessageLinktestReq(c.hp.getNextSystemId(systemId)).ToBytes())
	if err != nil {
		c.hp.logger.Println("send error : ", err)
		return
	}

	_, err = c.hp.systemQueues[systemId].Get(c.hp.timeouts.T6ControlTransTimeout())
	if err != nil {
		c.hp.logger.Println("timeout get rsp : ", err)
		return
	}
}

func (c *HsmsConnection) sendDeselectRsp(message ast.HSMSMessage, selectStatus byte) {
	c.connection.Send(ast.NewHSMSMessageDeselectRsp(message, selectStatus).ToBytes())
}

func (c *HsmsConnection) sendSelectRsp(message ast.HSMSMessage, selectStatus byte) {
	c.connection.Send(ast.NewHSMSMessageSelectRsp(message, selectStatus).ToBytes())
}

func (c *HsmsConnection) sendLinkTestRsp(message ast.HSMSMessage) {
	c.connection.Send(ast.NewHSMSMessageLinktestRsp(message).ToBytes())
}

func (c *HsmsConnection) sendReject(message ast.HSMSMessage, reasonCode byte) {
	c.connection.Send(ast.NewHSMSMessageRejectReqFromMsg(message, reasonCode).ToBytes())
}

func (c *HsmsConnection) sendSelectReq() (err error) {

	systemId := c.hp.getNextSystemCounter()
	c.hp.systemQueues[systemId] = utils.NewDeque()
	defer delete(c.hp.systemQueues, systemId)

	err = c.connection.Send(ast.NewHSMSMessageSelectReq(0xFFFF, c.hp.getNextSystemId(systemId)).ToBytes())
	if err != nil {
		c.hp.logger.Println("send error : ", err)
		return
	}

	_, err = c.hp.systemQueues[systemId].Get(c.hp.timeouts.T6ControlTransTimeout())
	if err != nil {
		c.hp.logger.Println("timeout get rsp : ", err)
		return
	}

	return nil
}
