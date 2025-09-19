package hsms

import (
	"fmt"

	link "github.com/younglifestyle/secs4go"
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
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

// HsmsConnection encapsulates active/passive connection behaviour shared by both roles.
type HsmsConnection struct {
	active        bool
	remoteAddress string
	remotePort    int
	sessionID     int
	hp            *HsmsProtocol
	connection    *link.Session
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
	if c.connection == nil {
		return ErrNotConnected
	}
	return c.connection.Send(msg)
}

func (c *HsmsConnection) Close() error {
	if c.connection == nil {
		return nil
	}
	return c.connection.Close()
}

func (c *HsmsConnection) sendRejectRsp(packet ast.HSMSMessage, reasonCode byte) {
	rejectReq := ast.NewHSMSMessageRejectReqFromMsg(packet, reasonCode)
	c.hp.logControlMessage("TX", rejectReq)
	if err := c.connection.Send(rejectReq.ToBytes()); err != nil {
		c.hp.logger.Println("send reject rsp error:", err)
	}
}

func (c *HsmsConnection) sendLinktestReq() {
	systemID := c.hp.getNextSystemCounter()
	queue := c.hp.createQueue(systemID)
	defer c.hp.removeQueue(systemID)

	message := ast.NewHSMSMessageLinktestReq(c.hp.encodeSystemID(systemID))
	c.hp.logControlMessage("TX", message)
	if err := c.connection.Send(message.ToBytes()); err != nil {
		c.hp.logger.Println("send linktest.req error:", err)
		return
	}

	if _, err := queue.Get(c.hp.timeouts.T6ControlTransTimeout()); err != nil {
		c.hp.logger.Println("timeout waiting linktest.rsp:", err)
	}
}

func (c *HsmsConnection) sendDeselectRsp(message ast.HSMSMessage, selectStatus byte) {
	response := ast.NewHSMSMessageDeselectRsp(message, selectStatus)
	c.hp.logControlMessage("TX", response)
	if err := c.connection.Send(response.ToBytes()); err != nil {
		c.hp.logger.Println("send deselect.rsp error:", err)
	}
}

func (c *HsmsConnection) sendSelectRsp(message ast.HSMSMessage, selectStatus byte) {
	response := ast.NewHSMSMessageSelectRsp(message, selectStatus)
	c.hp.logControlMessage("TX", response)
	if err := c.connection.Send(response.ToBytes()); err != nil {
		c.hp.logger.Println("send select.rsp error:", err)
	}
}

func (c *HsmsConnection) sendLinkTestRsp(message ast.HSMSMessage) {
	response := ast.NewHSMSMessageLinktestRsp(message)
	c.hp.logControlMessage("TX", response)
	if err := c.connection.Send(response.ToBytes()); err != nil {
		c.hp.logger.Println("send linktest.rsp error:", err)
	}
}

func (c *HsmsConnection) sendReject(message ast.HSMSMessage, reasonCode byte) {
	reject := ast.NewHSMSMessageRejectReqFromMsg(message, reasonCode)
	c.hp.logControlMessage("TX", reject)
	if err := c.connection.Send(reject.ToBytes()); err != nil {
		c.hp.logger.Println("send reject.req error:", err)
	}
}

func (c *HsmsConnection) sendSelectReq() error {
	systemID := c.hp.getNextSystemCounter()
	queue := c.hp.createQueue(systemID)
	defer c.hp.removeQueue(systemID)

	request := ast.NewHSMSMessageSelectReq(uint16(c.hp.sessionID), c.hp.encodeSystemID(systemID))
	c.hp.logControlMessage("TX", request)
	if err := c.connection.Send(request.ToBytes()); err != nil {
		c.hp.logger.Println("send select.req error:", err)
		return err
	}

	resp, err := queue.Get(c.hp.timeouts.T6ControlTransTimeout())
	if err != nil {
		c.hp.logger.Println("timeout waiting select.rsp:", err)
		return err
	}

	ctrl, ok := resp.(ast.HSMSMessage)
	if !ok {
		return fmt.Errorf("unexpected response type %T", resp)
	}
	controlMsg, ok := ctrl.(*ast.ControlMessage)
	if !ok {
		return fmt.Errorf("unexpected control response %T", ctrl)
	}
	if controlMsg.Status() != 0 {
		return fmt.Errorf("select.rsp returned status %d", controlMsg.Status())
	}

	return nil
}

func (c *HsmsConnection) sendDeselectReq() error {
	systemID := c.hp.getNextSystemCounter()
	queue := c.hp.createQueue(systemID)
	defer c.hp.removeQueue(systemID)

	request := ast.NewHSMSMessageDeselectReq(uint16(c.hp.sessionID), c.hp.encodeSystemID(systemID))
	c.hp.logControlMessage("TX", request)
	if err := c.connection.Send(request.ToBytes()); err != nil {
		c.hp.logger.Println("send deselect.req error:", err)
		return err
	}

	resp, err := queue.Get(c.hp.timeouts.T6ControlTransTimeout())
	if err != nil {
		c.hp.logger.Println("timeout waiting deselect.rsp:", err)
		return err
	}

	ctrl, ok := resp.(ast.HSMSMessage)
	if !ok {
		return fmt.Errorf("unexpected response type %T", resp)
	}
	controlMsg, ok := ctrl.(*ast.ControlMessage)
	if !ok {
		return fmt.Errorf("unexpected control response %T", ctrl)
	}
	if controlMsg.Status() != 0 {
		return fmt.Errorf("deselect.rsp returned status %d", controlMsg.Status())
	}

	return nil
}
