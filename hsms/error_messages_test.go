package hsms

import (
	"testing"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// TestBuildS9F1_UnrecognizedDeviceID verifies S9F1 message format
func TestBuildS9F1_UnrecognizedDeviceID(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x01}
	unrecognizedID := byte(0x42)

	msg := BuildS9F1(unrecognizedID, systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F1 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 1 {
		t.Errorf("Expected function 1, got %d", msg.FunctionCode())
	}

	// Verify body contains the unrecognized device ID
	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	binNode, ok := body.(*ast.BinaryNode)
	if !ok {
		t.Fatalf("Expected BinaryNode, got %T", body)
	}

	values, ok := binNode.Values().([]int)
	if !ok || len(values) != 1 {
		t.Fatalf("Expected single binary value, got %v", binNode.Values())
	}

	if values[0] != int(unrecognizedID) {
		t.Errorf("Expected device ID %d, got %d", unrecognizedID, values[0])
	}
}

// TestBuildS9F3_UnrecognizedStream verifies S9F3 message format
func TestBuildS9F3_UnrecognizedStream(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x02}
	unrecognizedStream := byte(99)

	msg := BuildS9F3(unrecognizedStream, systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F3 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 3 {
		t.Errorf("Expected function 3, got %d", msg.FunctionCode())
	}

	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	binNode, ok := body.(*ast.BinaryNode)
	if !ok {
		t.Fatalf("Expected BinaryNode, got %T", body)
	}

	values, ok := binNode.Values().([]int)
	if !ok || len(values) != 1 {
		t.Fatalf("Expected single binary value, got %v", binNode.Values())
	}

	if values[0] != int(unrecognizedStream) {
		t.Errorf("Expected stream %d, got %d", unrecognizedStream, values[0])
	}
}

// TestBuildS9F5_UnrecognizedFunction verifies S9F5 message format
func TestBuildS9F5_UnrecognizedFunction(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x03}
	unrecognizedFunction := byte(99)

	msg := BuildS9F5(unrecognizedFunction, systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F5 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 5 {
		t.Errorf("Expected function 5, got %d", msg.FunctionCode())
	}

	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	binNode, ok := body.(*ast.BinaryNode)
	if !ok {
		t.Fatalf("Expected BinaryNode, got %T", body)
	}

	values, ok := binNode.Values().([]int)
	if !ok || len(values) != 1 {
		t.Fatalf("Expected single binary value, got %v", binNode.Values())
	}

	if values[0] != int(unrecognizedFunction) {
		t.Errorf("Expected function %d, got %d", unrecognizedFunction, values[0])
	}
}

// TestBuildS9F7_IllegalData verifies S9F7 message format
func TestBuildS9F7_IllegalData(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x04}

	msg := BuildS9F7(systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F7 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 7 {
		t.Errorf("Expected function 7, got %d", msg.FunctionCode())
	}

	// S9F7 should have empty body per standard
	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	listNode, ok := body.(*ast.ListNode)
	if !ok {
		t.Fatalf("Expected ListNode, got %T", body)
	}

	if listNode.Size() != 0 {
		t.Errorf("Expected empty list, got size %d", listNode.Size())
	}
}

// TestBuildS9F9_TransactionTimeout verifies S9F9 message format
func TestBuildS9F9_TransactionTimeout(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x05}
	timedOutStream := byte(1)
	// BuildS9F9 expects a 10-byte SECS header, not a single byte
	header := []byte{0, 0, timedOutStream, 1, 0, 0, 0, 0, 0, 5}

	msg := BuildS9F9(header, systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F9 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 9 {
		t.Errorf("Expected function 9, got %d", msg.FunctionCode())
	}

	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	binNode, ok := body.(*ast.BinaryNode)
	if !ok {
		t.Fatalf("Expected BinaryNode, got %T", body)
	}

	values, ok := binNode.Values().([]int)
	if !ok {
		t.Fatalf("Expected []int, got %T", binNode.Values())
	}

	// BuildS9F9 now takes full 10-byte header, not just stream byte
	if len(values) != 10 {
		t.Errorf("Expected 10-byte header, got %d bytes: %v", len(values), values)
	}

	// Verify stream byte is in correct position (byte 2 of header)
	if values[2] != int(timedOutStream) {
		t.Errorf("Expected stream %d at position 2, got %d", timedOutStream, values[2])
	}
}

// TestBuildS9F11_DataTooLong verifies S9F11 message format
func TestBuildS9F11_DataTooLong(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x06}

	msg := BuildS9F11(systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F11 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 11 {
		t.Errorf("Expected function 11, got %d", msg.FunctionCode())
	}

	// S9F11 should have empty body per standard
	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	listNode, ok := body.(*ast.ListNode)
	if !ok {
		t.Fatalf("Expected ListNode, got %T", body)
	}

	if listNode.Size() != 0 {
		t.Errorf("Expected empty list, got size %d", listNode.Size())
	}
}

// TestBuildS9F13_ConversationTimeout verifies S9F13 message format
func TestBuildS9F13_ConversationTimeout(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x07}

	msg := BuildS9F13(systemBytes)

	if msg == nil {
		t.Fatal("BuildS9F13 returned nil")
	}

	if msg.StreamCode() != 9 {
		t.Errorf("Expected stream 9, got %d", msg.StreamCode())
	}

	if msg.FunctionCode() != 13 {
		t.Errorf("Expected function 13, got %d", msg.FunctionCode())
	}

	// S9F13 should have empty body per standard
	body, err := msg.Get()
	if err != nil {
		t.Fatalf("Failed to get message body: %v", err)
	}

	listNode, ok := body.(*ast.ListNode)
	if !ok {
		t.Fatalf("Expected ListNode, got %T", body)
	}

	if listNode.Size() != 0 {
		t.Errorf("Expected empty list, got size %d", listNode.Size())
	}
}

// TestS9ErrorInfo verifies S9ErrorInfo structure
func TestS9ErrorInfo(t *testing.T) {
	systemBytes := []byte{0x00, 0x00, 0x00, 0x08}
	msg := BuildS9F3(99, systemBytes)

	errorInfo := NewS9ErrorInfo(
		S9F3UnrecognizedStream,
		ErrTextUnrecognizedStream,
		msg,
		systemBytes,
	)

	if errorInfo == nil {
		t.Fatal("NewS9ErrorInfo returned nil")
	}

	if errorInfo.ErrorCode != S9F3UnrecognizedStream {
		t.Errorf("Expected error code %d, got %d", S9F3UnrecognizedStream, errorInfo.ErrorCode)
	}

	if errorInfo.ErrorText != ErrTextUnrecognizedStream {
		t.Errorf("Expected error text %q, got %q", ErrTextUnrecognizedStream, errorInfo.ErrorText)
	}

	if errorInfo.OriginalMessage != msg {
		t.Error("OriginalMessage not set correctly")
	}
}

// TestMaxHSMSMessageSize verifies the constant value
func TestMaxHSMSMessageSize(t *testing.T) {
	expected := 16 * 1024 * 1024
	if MaxHSMSMessageSize != expected {
		t.Errorf("Expected MaxHSMSMessageSize to be %d, got %d", expected, MaxHSMSMessageSize)
	}
}
