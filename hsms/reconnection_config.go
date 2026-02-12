package hsms

// Reconnection configuration getters and setters

// MaxReconnectAttempts returns the maximum number of reconnection attempts (0 = unlimited)
func (s *SecsTimeout) MaxReconnectAttempts() int {
	return s.maxReconnectAttempts
}

// SetMaxReconnectAttempts sets the maximum number of reconnection attempts (0 = unlimited)
func (s *SecsTimeout) SetMaxReconnectAttempts(max int) {
	s.maxReconnectAttempts = max
}

// ReconnectBackoffBase returns the base delay for exponential backoff (in seconds)
func (s *SecsTimeout) ReconnectBackoffBase() int {
	return s.reconnectBackoffBase
}

// SetReconnectBackoffBase sets the base delay for exponential backoff (in seconds)
func (s *SecsTimeout) SetReconnectBackoffBase(base int) {
	s.reconnectBackoffBase = base
}

// ReconnectBackoffMax returns the maximum delay for reconnection backoff (in seconds)
func (s *SecsTimeout) ReconnectBackoffMax() int {
	return s.reconnectBackoffMax
}

// SetReconnectBackoffMax sets the maximum delay for reconnection backoff (in seconds)
func (s *SecsTimeout) SetReconnectBackoffMax(max int) {
	s.reconnectBackoffMax = max
}

// AutoReconnect returns whether automatic reconnection is enabled
func (s *SecsTimeout) AutoReconnect() bool {
	return s.autoReconnect
}

// SetAutoReconnect sets whether automatic reconnection is enabled
func (s *SecsTimeout) SetAutoReconnect(enabled bool) {
	s.autoReconnect = enabled
}
