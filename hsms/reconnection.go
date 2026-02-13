package hsms

import (
	"math"
	"time"
)

// calculateBackoffDelay calculates the delay for exponential backoff
func (p *HsmsProtocol) calculateBackoffDelay() time.Duration {
	base := p.timeouts.ReconnectBackoffBase()
	max := p.timeouts.ReconnectBackoffMax()

	// Calculate: base * 2^(attempts-1)
	// For attempt 1: base * 2^0 = base
	// For attempt 2: base * 2^1 = base * 2
	// For attempt 3: base * 2^2 = base * 4
	delay := float64(base) * math.Pow(2, float64(p.reconnectAttempts-1))

	// Cap at maximum
	if delay > float64(max) {
		delay = float64(max)
	}

	return time.Duration(delay) * time.Second
}

// resetReconnectionState resets the reconnection attempt counter
func (p *HsmsProtocol) resetReconnectionState() {
	p.reconnectAttempts = 0
	p.lastDisconnectTime = time.Time{}
}

// onDisconnection handles disconnection events and prepares for reconnection
func (p *HsmsProtocol) onDisconnection() {
	p.lastDisconnectTime = time.Now()
	// Don't reset reconnectAttempts here - let it accumulate for backoff
	p.logger.Info("disconnected", "time", p.lastDisconnectTime.Format(time.RFC3339), "reconnectAttempts", p.reconnectAttempts)
}
