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
		c.hp.logger.Error("send reject rsp failed", "error", err)
	}
}

func (c *HsmsConnection) sendLinktestReq() {
	systemID := c.hp.getNextSystemCounter()
	queue := c.hp.createQueue(systemID)
	defer c.hp.removeQueue(systemID)

	message := ast.NewHSMSMessageLinktestReq(c.hp.encodeSystemID(systemID))
	c.hp.logControlMessage("TX", message)
	if err := c.connection.Send(message.ToBytes()); err != nil {
		c.hp.logger.Error("send linktest.req failed", "error", err)
		return
	}

	if _, err := queue.Get(c.hp.timeouts.T6ControlTransTimeout()); err != nil {
		c.hp.logger.Warn("timeout waiting linktest.rsp", "error", err)
	}
}

func (c *HsmsConnection) sendDeselectRsp(message ast.HSMSMessage, status ControlStatus) {
	response := ast.NewHSMSMessageDeselectRsp(message, byte(status))
	c.hp.logControlMessage("TX", response)
	if err := c.connection.Send(response.ToBytes()); err != nil {
		c.hp.logger.Error("send deselect.rsp failed", "error", err)
	}
}

func (c *HsmsConnection) sendSelectRsp(message ast.HSMSMessage, status ControlStatus) {
	response := ast.NewHSMSMessageSelectRsp(message, byte(status))
	c.hp.logControlMessage("TX", response)
	if err := c.connection.Send(response.ToBytes()); err != nil {
		c.hp.logger.Error("send select.rsp failed", "error", err)
	}
}

func (c *HsmsConnection) sendLinkTestRsp(message ast.HSMSMessage) {
	response := ast.NewHSMSMessageLinktestRsp(message)
	c.hp.logControlMessage("TX", response)
	if err := c.connection.Send(response.ToBytes()); err != nil {
		c.hp.logger.Error("send linktest.rsp failed", "error", err)
	}
}

func (c *HsmsConnection) sendReject(message ast.HSMSMessage, reason RejectReason) {
	reject := ast.NewHSMSMessageRejectReqFromMsg(message, byte(reason))
	c.hp.logControlMessage("TX", reject)
	if err := c.connection.Send(reject.ToBytes()); err != nil {
		c.hp.logger.Error("send reject.req failed", "error", err)
	}
}

func (c *HsmsConnection) sendSelectReq() error {
	systemID := c.hp.getNextSystemCounter()
	queue := c.hp.createQueue(systemID)
	defer c.hp.removeQueue(systemID)

	request := ast.NewHSMSMessageSelectReq(uint16(c.hp.sessionID), c.hp.encodeSystemID(systemID))
	c.hp.logControlMessage("TX", request)
	if err := c.connection.Send(request.ToBytes()); err != nil {
		c.hp.logger.Error("send select.req failed", "error", err)
		return err
	}

	resp, err := queue.Get(c.hp.timeouts.T6ControlTransTimeout())
	if err != nil {
		c.hp.logger.Warn("timeout waiting select.rsp", "error", err)
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
	if ControlStatus(controlMsg.Status()) != ControlStatusAccepted {
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
		c.hp.logger.Error("send deselect.req failed", "error", err)
		return err
	}

	resp, err := queue.Get(c.hp.timeouts.T6ControlTransTimeout())
	if err != nil {
		c.hp.logger.Warn("timeout waiting deselect.rsp", "error", err)
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
	if ControlStatus(controlMsg.Status()) != ControlStatusAccepted {
		return fmt.Errorf("deselect.rsp returned status %d", controlMsg.Status())
	}

	return nil
}
