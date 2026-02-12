package gem

import (
	"errors"
	"sync"
	"time"
)

// TimeProvider is a callback function that returns the current equipment time.
// If not set, uses system time.
type TimeProvider func() time.Time

// ClockSyncHandler is called when the host requests to set equipment time via S2F31.
// It should validate the requested time and return TIACK code:
//   - 0: Accepted
//   - 1: Not allowed now
//   - 2: Out of synchronization limit
type ClockSyncHandler func(requestedTime time.Time) (tiack byte, err error)

// ClockManager handles time/clock related operations for GEM.
type ClockManager struct {
	timeProvider     TimeProvider
	clockSyncHandler ClockSyncHandler
	mu               sync.RWMutex
}

// NewClockManager creates a new clock manager with default system time provider.
func NewClockManager() *ClockManager {
	return &ClockManager{
		timeProvider: time.Now,
	}
}

// SetTimeProvider sets a custom time provider for equipment.
// This is useful for testing or equipment with custom time sources.
func (c *ClockManager) SetTimeProvider(provider TimeProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeProvider = provider
}

// SetClockSyncHandler sets the handler for S2F31 time set requests.
// If not set, S2F31 will return TIACK=1 (not allowed).
func (c *ClockManager) SetClockSyncHandler(handler ClockSyncHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clockSyncHandler = handler
}

// GetTime returns the current equipment time using the configured provider.
func (c *ClockManager) GetTime() time.Time {
	c.mu.RLock()
	provider := c.timeProvider
	c.mu.RUnlock()

	if provider == nil {
		return time.Now()
	}
	return provider()
}

// GetFormattedTime returns equipment time in SEMI E5 format: "YYYYMMDDhhmmss00"
// The last two digits are centiseconds (always 00 in this implementation).
func (c *ClockManager) GetFormattedTime() string {
	t := c.GetTime()
	// Format: YYYYMMDDhhmmss00
	return t.Format("20060102150405") + "00"
}

// ParseSEMITime parses a SEMI E5 format time string ("YYYYMMDDhhmmss00").
func ParseSEMITime(timeStr string) (time.Time, error) {
	if len(timeStr) < 14 {
		return time.Time{}, errors.New("gem: invalid SEMI time format")
	}
	// Parse first 14 characters (ignore centiseconds)
	return time.Parse("20060102150405", timeStr[:14])
}

// HandleTimeSet processes S2F31 time set request.
// Returns TIACK code.
func (c *ClockManager) HandleTimeSet(requestedTime time.Time) byte {
	c.mu.RLock()
	handler := c.clockSyncHandler
	c.mu.RUnlock()

	if handler == nil {
		// No handler configured, reject request
		return 1 // Not allowed
	}

	tiack, err := handler(requestedTime)
	if err != nil {
		// Handler error, return "not allowed"
		return 1
	}

	return tiack
}
