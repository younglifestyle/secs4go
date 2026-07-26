package hsms

import (
	"fmt"
	"unicode/utf8"

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
// Message structure: MHEAD BINARY[10] - the header of the rejected message.
func BuildS9F1(messageHeader interface{}, systemBytes []byte) *ast.DataMessage {
	body := binaryHeaderNode(normalizeLegacyHeader(messageHeader, 1))
	return ast.NewHSMSDataMessage("", 9, 1, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F3 creates an S9F3 message (Unrecognized Stream).
// This is sent when a message is received with a stream code that is not supported.
//
// Message structure: MHEAD BINARY[10] - the header of the rejected message.
func BuildS9F3(messageHeader interface{}, systemBytes []byte) *ast.DataMessage {
	body := binaryHeaderNode(normalizeLegacyHeader(messageHeader, 2))
	return ast.NewHSMSDataMessage("", 9, 3, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F5 creates an S9F5 message (Unrecognized Function).
// This is sent when a message is received with a function code that is not supported
// for the given stream.
//
// Message structure: MHEAD BINARY[10] - the header of the rejected message.
func BuildS9F5(messageHeader interface{}, systemBytes []byte) *ast.DataMessage {
	body := binaryHeaderNode(normalizeLegacyHeader(messageHeader, 3))
	return ast.NewHSMSDataMessage("", 9, 5, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F7 creates an S9F7 message (Illegal Data).
// This is sent when a message is received with data that cannot be parsed or
// does not conform to the expected format.
//
// Message structure: MHEAD BINARY[10]. The optional argument is the offending
// message header; omitting it produces an all-zero header for API compatibility.
func BuildS9F7(systemBytes []byte, messageHeader ...[]byte) *ast.DataMessage {
	body := binaryHeaderNode(firstHeader(messageHeader))
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
	body := binaryHeaderNode(sHeader)
	return ast.NewHSMSDataMessage("", 9, 9, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F11 creates an S9F11 message (Data Too Long).
// This is sent when a message exceeds the maximum allowable size.
//
// Message structure: MHEAD BINARY[10]. The optional argument is the offending
// message header; omitting it produces an all-zero header for API compatibility.
func BuildS9F11(systemBytes []byte, messageHeader ...[]byte) *ast.DataMessage {
	body := binaryHeaderNode(firstHeader(messageHeader))
	return ast.NewHSMSDataMessage("", 9, 11, 0, "H<-E", body, 0, systemBytes)
}

func firstHeader(headers [][]byte) []byte {
	if len(headers) == 0 {
		return nil
	}
	return headers[0]
}

func binaryHeaderNode(header []byte) ast.ItemNode {
	values := make([]interface{}, 10)
	for i := range values {
		if i < len(header) {
			values[i] = header[i]
		} else {
			values[i] = byte(0)
		}
	}
	return ast.NewBinaryNode(values...)
}

func normalizeLegacyHeader(value interface{}, legacyIndex int) []byte {
	if header, ok := value.([]byte); ok {
		return header
	}
	header := make([]byte, 10)
	switch v := value.(type) {
	case byte:
		header[legacyIndex] = v
	case int:
		header[legacyIndex] = byte(v)
	}
	return header
}

// BuildS9F13 creates a legacy empty S9F13 message.
// Deprecated: use BuildS9F13ConversationTimeout to include required MEXP/EDID data.
func BuildS9F13(systemBytes []byte) *ast.DataMessage {
	body := ast.NewListNode()
	return ast.NewHSMSDataMessage("", 9, 13, 0, "H<-E", body, 0, systemBytes)
}

// BuildS9F13ConversationTimeout creates a standards-shaped S9F13 body:
// <L <A[6] MEXP> <EDID>>.
func BuildS9F13ConversationTimeout(messageExpected string, entityID ast.ItemNode, systemBytes []byte) (*ast.DataMessage, error) {
	if !utf8.ValidString(messageExpected) || len(messageExpected) != 6 {
		return nil, fmt.Errorf("hsms: MEXP must contain exactly 6 ASCII bytes")
	}
	for _, value := range []byte(messageExpected) {
		if value > 0x7F {
			return nil, fmt.Errorf("hsms: MEXP must contain only ASCII characters")
		}
	}
	if entityID == nil {
		return nil, fmt.Errorf("hsms: EDID is required")
	}
	body := ast.NewListNode(ast.NewASCIINode(messageExpected), entityID)
	return ast.NewHSMSDataMessage("", 9, 13, 0, "H<-E", body, 0, systemBytes), nil
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
	// MessageHeader is the 10-byte MHEAD/SHEAD carried by the S9 message.
	MessageHeader []byte
}

// NewS9ErrorInfo creates a new S9ErrorInfo with the given parameters.
func NewS9ErrorInfo(errorCode int, errorText string, originalMsg *ast.DataMessage, systemBytes []byte) *S9ErrorInfo {
	systemBytesCopy := append([]byte(nil), systemBytes...)
	return &S9ErrorInfo{
		ErrorCode:       errorCode,
		ErrorText:       errorText,
		OriginalMessage: originalMsg,
		SystemBytes:     systemBytesCopy,
	}
}
