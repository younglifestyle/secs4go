package hsms

import (
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// sendS9ErrorForUnrecognized sends appropriate S9 error message for unrecognized stream/function
func (p *HsmsProtocol) sendS9ErrorForUnrecognized(message *ast.DataMessage) {
	stream := message.StreamCode()
	function := message.FunctionCode()
	systemBytes := message.SystemBytes()

	// Determine if stream is recognized (streams 1-21 are defined in SEMI E5)
	// For now, we consider streams 1-7, 9, 10, 12-17 as potentially valid
	// Stream 9 is for errors, so we won't send S9 about S9
	if stream == 9 {
		p.logger.Printf("received Stream 9 message S9F%d - not sending S9 response", function)
		return
	}

	var s9Message *ast.DataMessage

	// Check if it's an unknown stream entirely
	isKnownStream := p.isKnownStream(stream)

	if !isKnownStream {
		// Unknown stream - send S9F3
		s9Message = BuildS9F3(byte(stream), systemBytes)
		p.logger.Printf("unrecognized stream S%dF%d - sending S9F3", stream, function)
	} else {
		// Known stream but unknown function - send S9F5
		s9Message = BuildS9F5(byte(function), systemBytes)
		p.logger.Printf("unrecognized function S%dF%d - sending S9F5", stream, function)
	}

	if s9Message != nil {
		// Set session ID and ensure no wait bit
		s9Message = s9Message.SetSessionIDAndSystemBytes(p.sessionID, systemBytes)
		s9Message = s9Message.SetWaitBit(false)

		p.logDataMessage("OUT", s9Message)

		if err := p.hsmsConnection.Send(s9Message.ToBytes()); err != nil {
			p.logger.Printf("failed to send S9 error message: %v", err)
		}
	}
}

// isKnownStream checks if a stream code is recognized by the protocol
// This is a basic implementation - in a full system, this would check against
// registered handlers or a configuration
func (p *HsmsProtocol) isKnownStream(stream int) bool {
	// SEMI E5 defines the following streams as standard:
	// 1: Equipment Status
	// 2: Equipment Control
	// 3: Material Status
	// 4: Material Control
	// 5: Exception Handling (Alarms)
	// 6: Data Collection
	// 7: Process Program Management
	// 8: Control Program Transfer
	// 9: System Errors (handled specially)
	// 10: Terminal Services
	// 12: Wafer Mapping
	// 13: Data Set Transfer
	// 14: Object Services
	// 15: Recipe Management
	// 16: Processing Management
	// 17: Operation Management
	// 21: Segmented Data Transfer

	knownStreams := map[int]bool{
		1: true, 2: true, 3: true, 4: true, 5: true,
		6: true, 7: true, 8: true, 9: true, 10: true,
		12: true, 13: true, 14: true, 15: true, 16: true,
		17: true, 21: true,
	}

	return knownStreams[stream]
}

// sendS9F7IllegalData sends S9F7 for illegal data format
func (p *HsmsProtocol) sendS9F7IllegalData(systemBytes []byte) {
	s9Message := BuildS9F7(systemBytes)
	s9Message = s9Message.SetSessionIDAndSystemBytes(p.sessionID, systemBytes)
	s9Message = s9Message.SetWaitBit(false)

	p.logger.Printf("illegal data format - sending S9F7")
	p.logDataMessage("OUT", s9Message)

	if err := p.hsmsConnection.Send(s9Message.ToBytes()); err != nil {
		p.logger.Printf("failed to send S9F7: %v", err)
	}
}

// sendS9F11DataTooLong sends S9F11 for oversized messages
func (p *HsmsProtocol) sendS9F11DataTooLong(systemBytes []byte) {
	s9Message := BuildS9F11(systemBytes)
	s9Message = s9Message.SetSessionIDAndSystemBytes(p.sessionID, systemBytes)
	s9Message = s9Message.SetWaitBit(false)

	p.logger.Printf("message too long - sending S9F11")
	p.logDataMessage("OUT", s9Message)

	if err := p.hsmsConnection.Send(s9Message.ToBytes()); err != nil {
		p.logger.Printf("failed to send S9F11: %v", err)
	}
}

// sendS9F9TransactionTimeout sends S9F9 for T3 transaction timeout
func (p *HsmsProtocol) sendS9F9TransactionTimeout(originalMsg *ast.DataMessage) {
	// S9F9: Transaction Timer Timeout
	// W: <B [10] SHeader>
	// The stored header of the message associated with the transaction timer timeout.

	// We need to reconstruct the 10-byte header from the original message
	// Device ID (2 bytes), Stream (1 byte), Function (1 byte), PType (1 byte), SType (1 byte), System Bytes (4 bytes)

	headerBytes := make([]byte, 10)

	// Device ID / Session ID
	session := originalMsg.SessionID()
	headerBytes[0] = byte(session >> 8)
	headerBytes[1] = byte(session)

	// Stream & Function - Wait Bit is in Stream byte
	stream := byte(originalMsg.StreamCode())
	if originalMsg.WaitBit() == "true" {
		stream |= 0x80
	}
	headerBytes[2] = stream
	headerBytes[3] = byte(originalMsg.FunctionCode())

	// PType, SType
	headerBytes[4] = 0 // PType = 0 (SECS-II)
	headerBytes[5] = 0 // SType = 0 (Data Message)

	// System Bytes
	sysBytes := originalMsg.SystemBytes()
	if len(sysBytes) >= 4 {
		copy(headerBytes[6:], sysBytes[:4])
	}

	s9Message := BuildS9F9(headerBytes, nil)

	if s9Message != nil {
		// Set session ID and ensure no wait bit for error message
		// Use next system bytes for the S9 message itself
		systemID := p.getNextSystemCounter()
		nextSys := p.encodeSystemID(systemID)

		s9Message = s9Message.SetSessionIDAndSystemBytes(p.sessionID, nextSys)
		s9Message = s9Message.SetWaitBit(false)

		p.logger.Printf("T3 timeout for S%dF%d - sending S9F9", originalMsg.StreamCode(), originalMsg.FunctionCode())
		p.logDataMessage("OUT", s9Message)

		if err := p.hsmsConnection.Send(s9Message.ToBytes()); err != nil {
			p.logger.Printf("failed to send S9F9: %v", err)
		}
	}
}
