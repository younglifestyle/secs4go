package hsms

import (
	"testing"
	"time"
)

// TestReconnectionConfiguration tests reconnection config getter/setters
func TestReconnectionConfiguration(t *testing.T) {
	timeout := NewSecsTimeout()

	// Test defaults
	if timeout.MaxReconnectAttempts() != 10 {
		t.Errorf("expected default max reconnect attempts 10, got %d", timeout.MaxReconnectAttempts())
	}

	if timeout.ReconnectBackoffBase() != 2 {
		t.Errorf("expected default backoff base 2, got %d", timeout.ReconnectBackoffBase())
	}

	if timeout.ReconnectBackoffMax() != 60 {
		t.Errorf("expected default backoff max 60, got %d", timeout.ReconnectBackoffMax())
	}

	if !timeout.AutoReconnect() {
		t.Error("expected auto-reconnect enabled by default")
	}

	// Test setters
	timeout.SetMaxReconnectAttempts(5)
	if timeout.MaxReconnectAttempts() != 5 {
		t.Errorf("expected max reconnect attempts 5 after set, got %d", timeout.MaxReconnectAttempts())
	}

	timeout.SetReconnectBackoffBase(3)
	if timeout.ReconnectBackoffBase() != 3 {
		t.Errorf("expected backoff base 3 after set, got %d", timeout.ReconnectBackoffBase())
	}

	timeout.SetReconnectBackoffMax(120)
	if timeout.ReconnectBackoffMax() != 120 {
		t.Errorf("expected backoff max 120 after set, got %d", timeout.ReconnectBackoffMax())
	}

	timeout.SetAutoReconnect(false)
	if timeout.AutoReconnect() {
		t.Error("expected auto-reconnect disabled after set")
	}

	t.Log("✓ Reconnection configuration working correctly")
}

// TestBackoffCalculation tests exponential backoff delay calculation
func TestBackoffCalculation(t *testing.T) {
	protocol := NewHsmsProtocol("127.0.0.1", 5000, true, 0x0100, "test-backoff")

	testCases := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 2 * time.Second},   // 2 * 2^0 = 2
		{2, 4 * time.Second},   // 2 * 2^1 = 4
		{3, 8 * time.Second},   // 2 * 2^2 = 8
		{4, 16 * time.Second},  // 2 * 2^3 = 16
		{5, 32 * time.Second},  // 2 * 2^4 = 32
		{6, 60 * time.Second},  // 2 * 2^5 = 64, capped at 60
		{7, 60 * time.Second},  // Capped at max
		{10, 60 * time.Second}, // Capped at max
	}

	for _, tc := range testCases {
		protocol.reconnectAttempts = tc.attempt
		delay := protocol.calculateBackoffDelay()
		if delay != tc.expected {
			t.Errorf("attempt %d: expected delay %v, got %v", tc.attempt, tc.expected, delay)
		}
	}

	t.Log("✓ Exponential backoff calculation correct")
}

// TestBackoffWithCustomConfig tests backoff with custom configuration
func TestBackoffWithCustomConfig(t *testing.T) {
	protocol := NewHsmsProtocol("127.0.0.1", 5000, true, 0x0100, "test-custom-backoff")

	// Set custom backoff parameters
	protocol.timeouts.SetReconnectBackoffBase(5)
	protocol.timeouts.SetReconnectBackoffMax(100)

	testCases := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 5 * time.Second},   // 5 * 2^0 = 5
		{2, 10 * time.Second},  // 5 * 2^1 = 10
		{3, 20 * time.Second},  // 5 * 2^2 = 20
		{4, 40 * time.Second},  // 5 * 2^3 = 40
		{5, 80 * time.Second},  // 5 * 2^4 = 80
		{6, 100 * time.Second}, // 5 * 2^5 = 160, capped at 100
		{7, 100 * time.Second}, // Capped at max
	}

	for _, tc := range testCases {
		protocol.reconnectAttempts = tc.attempt
		delay := protocol.calculateBackoffDelay()
		if delay != tc.expected {
			t.Errorf("attempt %d: expected delay %v, got %v", tc.attempt, tc.expected, delay)
		}
	}

	t.Log("✓ Custom backoff configuration works correctly")
}

// TestReconnectionStateReset tests reconnection state reset
func TestReconnectionStateReset(t *testing.T) {
	protocol := NewHsmsProtocol("127.0.0.1", 5000, true, 0x0100, "test-reset")

	// Simulate some reconnection attempts
	protocol.reconnectAttempts = 5
	protocol.lastDisconnectTime = time.Now()

	// Reset state
	protocol.resetReconnectionState()

	if protocol.reconnectAttempts != 0 {
		t.Errorf("expected reconnect attempts to be 0 after reset, got %d", protocol.reconnectAttempts)
	}

	if !protocol.lastDisconnectTime.IsZero() {
		t.Error("expected lastDisconnectTime to be zero after reset")
	}

	t.Log("✓ Reconnection state reset works correctly")
}

// TestOnDisconnection tests disconnection event tracking
func TestOnDisconnection(t *testing.T) {
	protocol := NewHsmsProtocol("127.0.0.1", 5000, true, 0x0100, "test-disconnect")

	before := time.Now()
	protocol.onDisconnection()
	after := time.Now()

	if protocol.lastDisconnectTime.Before(before) || protocol.lastDisconnectTime.After(after) {
		t.Error("lastDisconnectTime not set correctly")
	}

	t.Log("✓ onDisconnection tracks disconnect time correctly")
}
