package hsms

import (
	"testing"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func TestBuildS9F1(t *testing.T) {
	systemBytes := []byte{0, 1, 2, 3}
	unrecognizedID := byte(0xFF)

	msg := BuildS9F1(unrecognizedID, systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 1 {
		t.Errorf("Expected S9F1, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	// Verify system bytes are set correctly
	if string(msg.SystemBytes()) != string(systemBytes) {
		t.Errorf("Expected SystemBytes %v, got %v", systemBytes, msg.SystemBytes())
	}

	// Verify message can be converted to bytes (is HSMS-ready)
	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}
}

func TestBuildS9F3(t *testing.T) {
	systemBytes := []byte{0, 1, 2, 3}
	unrecognizedStream := byte(100)

	msg := BuildS9F3(unrecognizedStream, systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 3 {
		t.Errorf("Expected S9F3, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}
}

func TestBuildS9F5(t *testing.T) {
	systemBytes := []byte{10, 11, 12, 13}
	unrecognizedFunc := byte(99)

	msg := BuildS9F5(unrecognizedFunc, systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 5 {
		t.Errorf("Expected S9F5, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}
}

func TestBuildS9F7(t *testing.T) {
	systemBytes := []byte{1, 1, 1, 1}

	// BuildS9F7 takes only systemBytes (has empty body)
	msg := BuildS9F7(systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 7 {
		t.Errorf("Expected S9F7, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}

	// S9F7 has empty list body
	body, err := msg.Get()
	if err != nil {
		t.Errorf("Failed to get body: %v", err)
	}
	if _, ok := body.(*ast.ListNode); !ok {
		t.Errorf("Expected ListNode body for S9F7, got %T", body)
	}
}

func TestBuildS9F9(t *testing.T) {
	header := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	systemBytes := []byte{0, 0, 0, 1}

	msg := BuildS9F9(header, systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 9 {
		t.Errorf("Expected S9F9, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	if string(msg.SystemBytes()) != string(systemBytes) {
		t.Errorf("Expected SystemBytes %v, got %v", systemBytes, msg.SystemBytes())
	}

	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}
}

func TestBuildS9F11(t *testing.T) {
	systemBytes := []byte{3, 3, 3, 3}

	msg := BuildS9F11(systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 11 {
		t.Errorf("Expected S9F11, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}

	// S9F11 has empty list body
	body, err := msg.Get()
	if err != nil {
		t.Errorf("Failed to get body: %v", err)
	}
	if _, ok := body.(*ast.ListNode); !ok {
		t.Errorf("Expected ListNode body for S9F11, got %T", body)
	}
}

func TestBuildS9F13(t *testing.T) {
	systemBytes := []byte{4, 4, 4, 4}

	msg := BuildS9F13(systemBytes)

	if msg.StreamCode() != 9 || msg.FunctionCode() != 13 {
		t.Errorf("Expected S9F13, got S%dF%d", msg.StreamCode(), msg.FunctionCode())
	}

	bytes := msg.ToBytes()
	if len(bytes) == 0 {
		t.Error("Message should be convertible to HSMS bytes")
	}

	// S9F13 has empty list body
	body, err := msg.Get()
	if err != nil {
		t.Errorf("Failed to get body: %v", err)
	}
	if _, ok := body.(*ast.ListNode); !ok {
		t.Errorf("Expected ListNode body for S9F13, got %T", body)
	}
}
