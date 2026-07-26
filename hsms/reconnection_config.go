package hsms

// Reconnection configuration getters and setters

// MaxReconnectAttempts returns the maximum number of reconnection attempts (0 = unlimited)
func (s *SecsTimeout) MaxReconnectAttempts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxReconnectAttempts
}

// SetMaxReconnectAttempts sets the maximum number of reconnection attempts (0 = unlimited)
func (s *SecsTimeout) SetMaxReconnectAttempts(max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxReconnectAttempts = max
}

// ReconnectBackoffBase returns the base delay for exponential backoff (in seconds)
func (s *SecsTimeout) ReconnectBackoffBase() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reconnectBackoffBase
}

// SetReconnectBackoffBase sets the base delay for exponential backoff (in seconds)
func (s *SecsTimeout) SetReconnectBackoffBase(base int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconnectBackoffBase = base
}

// ReconnectBackoffMax returns the maximum delay for reconnection backoff (in seconds)
func (s *SecsTimeout) ReconnectBackoffMax() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reconnectBackoffMax
}

// SetReconnectBackoffMax sets the maximum delay for reconnection backoff (in seconds)
func (s *SecsTimeout) SetReconnectBackoffMax(max int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconnectBackoffMax = max
}

// AutoReconnect returns whether automatic reconnection is enabled
func (s *SecsTimeout) AutoReconnect() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autoReconnect
}

// SetAutoReconnect sets whether automatic reconnection is enabled
func (s *SecsTimeout) SetAutoReconnect(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoReconnect = enabled
}
