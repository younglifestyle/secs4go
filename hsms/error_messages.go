package hsms

import (
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// Stream 9 Error Codes
// These constants define the various error types that can occur in SECS message processing.
const (
	// S9F1 - Unrecognized Device ID
	S9F1UnrecognizedDeviceID = 1

	// S9F3 - Unrecognized Stream Type
	S9F3UnrecognizedStream = 3

	// S9F5 - Unrecognized Function Type
	S9F5UnrecognizedFunction = 5

	// S9F7 - Illegal Data
	S9F7IllegalData = 7

	// S9F9 - Transaction Timer Timeout
	S9F9TransactionTimeout = 9

	// S9F11 - Data Too Long
	S9F11DataTooLong = 11

	// S9F13 - Conversation Timeout
	S9F13ConversationTimeout = 13
)

// Error text constants for logging and debugging
const (
	ErrTextUnrecognizedDeviceID = "Unrecognized Device ID"
	ErrTextUnrecognizedStream   = "Unrecognized Stream"
	ErrTextUnrecognizedFunction = "Unrecognized Function"
	ErrTextIllegalData          = "Illegal Data"
	ErrTextTransactionTimeout   = "Transaction Timer Timeout"
	ErrTextDataTooLong          = "Data Too Long"
	ErrTextConversationTimeout  = "Conversation Timeout"
)

// Maximum message size for HSMS (16MB based on SEMI E37 standard)
const MaxHSMSMessageSize = 16 * 1024 * 1024

// BuildS9F1 creates an S9F1 message (Unrecognized Device ID).
// This is sent when a message is received with a device ID that is not recognized.
//
// Message structure: BINARY[1] - the unrecognized device ID
func BuildS9F1(unrecognizedDeviceID byte, systemBytes []byte) *ast.DataMessage {
	body := ast.NewBinaryNode(int(unrecognizedDeviceID))
	return ast.NewHSMSDataMessage("", 9, 1, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F3 creates an S9F3 message (Unrecognized Stream).
// This is sent when a message is received with a stream code that is not supported.
//
// Message structure: BINARY[1] - the unrecognized stream code
func BuildS9F3(unrecognizedStream byte, systemBytes []byte) *ast.DataMessage {
	body := ast.NewBinaryNode(int(unrecognizedStream))
	return ast.NewHSMSDataMessage("", 9, 3, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F5 creates an S9F5 message (Unrecognized Function).
// This is sent when a message is received with a function code that is not supported
// for the given stream.
//
// Message structure: BINARY[1] - the unrecognized function code
func BuildS9F5(unrecognizedFunction byte, systemBytes []byte) *ast.DataMessage {
	body := ast.NewBinaryNode(int(unrecognizedFunction))
	return ast.NewHSMSDataMessage("", 9, 5, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F7 creates an S9F7 message (Illegal Data).
// This is sent when a message is received with data that cannot be parsed or
// does not conform to the expected format.
//
// Message structure: <empty> or optionally error description
func BuildS9F7(systemBytes []byte) *ast.DataMessage {
	// Per SEMI E5 standard, S9F7 can have empty body or ASCII error description
	// Using empty body for simplicity
	body := ast.NewListNode()
	return ast.NewHSMSDataMessage("", 9, 7, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F9 creates an S9F9 message (Transaction Timer Timeout).
// This is sent when a reply to a message is not received within the T3 timeout period.
//
// BuildS9F9 creates an S9F9 message (Transaction Timer Timeout).
// This is sent when a reply to a message is not received within the T3 timeout period.
//
// Message structure: BINARY[10] - the message header of the timed-out message
func BuildS9F9(sHeader []byte, systemBytes []byte) *ast.DataMessage {
	headerData := make([]interface{}, len(sHeader))
	for i, b := range sHeader {
		headerData[i] = b
	}
	body := ast.NewBinaryNode(headerData...)
	return ast.NewHSMSDataMessage("", 9, 9, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F11 creates an S9F11 message (Data Too Long).
// This is sent when a message exceeds the maximum allowable size.
//
// Message structure: <empty>
func BuildS9F11(systemBytes []byte) *ast.DataMessage {
	body := ast.NewListNode()
	return ast.NewHSMSDataMessage("", 9, 11, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F13 creates an S9F13 message (Conversation Timeout).
// This is sent when a multi-block transaction does not complete within
// the allowed time period.
//
// Message structure: <empty>
func BuildS9F13(systemBytes []byte) *ast.DataMessage {
	body := ast.NewListNode()
	return ast.NewHSMSDataMessage("", 9, 13, 0, "H<-E", body, 0, systemBytes)
}

// S9ErrorInfo contains information about a Stream 9 error that occurred.
type S9ErrorInfo struct {
	// ErrorCode is the S9 function code (1, 3, 5, 7, 9, 11, or 13)
	ErrorCode int
	// ErrorText is a human-readable description of the error
	ErrorText string
	// OriginalMessage is the message that caused the error (if available)
	OriginalMessage *ast.DataMessage
	// SystemBytes from the original message
	SystemBytes []byte
}

// NewS9ErrorInfo creates a new S9ErrorInfo with the given parameters.
func NewS9ErrorInfo(errorCode int, errorText string, originalMsg *ast.DataMessage, systemBytes []byte) *S9ErrorInfo {
	return &S9ErrorInfo{
		ErrorCode:       errorCode,
		ErrorText:       errorText,
		OriginalMessage: originalMsg,
		SystemBytes:     systemBytes,
	}
}
