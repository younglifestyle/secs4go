package gem

import (
	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// Clock synchronization handlers (S2F17/F18, S2F31/F32) - Equipment side

// onS2F17 handles Date and Time Request (Equipment side).
// Host → Equipment: S2F17 (empty)
// Equipment → Host: S2F18 W TIME (A[16] - "YYYYMMDDhhmmss00")
func (g *GemHandler) onS2F17(msg *ast.DataMessage) (*ast.DataMessage, error) {
	// Get formatted time from clock manager
	timeStr := g.clockManager.GetFormattedTime()

	// Build S2F18 response
	body := ast.NewASCIINode(timeStr)
	return ast.NewDataMessage("DateTimeData", 2, 18, 0, "E->H", body), nil
}

// onS2F31 handles Date and Time Set Request (Equipment side).
// Host → Equipment: S2F31 W TIME (A[16])
// Equipment → Host: S2F32 TIACK (BINARY[1] - 0=Accepted, 1-63=Error)
func (g *GemHandler) onS2F31(msg *ast.DataMessage) (*ast.DataMessage, error) {
	// Parse TIME from S2F31
	timeNode, err := msg.Get()
	if err != nil {
		// Malformed message, return TIACK=1
		return g.buildS2F32(1), nil
	}

	asciiNode, ok := timeNode.(*ast.ASCIINode)
	if !ok {
		return g.buildS2F32(1), nil
	}

	timeStr, ok := asciiNode.Values().(string)
	if !ok || len(timeStr) < 14 {
		return g.buildS2F32(1), nil
	}

	// Parse SEMI time format
	requestedTime, err := ParseSEMITime(timeStr)
	if err != nil {
		return g.buildS2F32(1), nil
	}

	// Call clock manager to handle time set
	tiack := g.clockManager.HandleTimeSet(requestedTime)

	return g.buildS2F32(tiack), nil
}

// buildS2F32 creates S2F32 response with TIACK code.
func (g *GemHandler) buildS2F32(tiack byte) *ast.DataMessage {
	body := ast.NewBinaryNode(int(tiack))
	return ast.NewDataMessage("DateTimeSetAck", 2, 32, 0, "E->H", body)
}

// Public APIs for clock management

// SetTimeProvider sets a custom time provider for equipment.
// This is useful for testing or equipment with custom time sources.
func (g *GemHandler) SetTimeProvider(provider TimeProvider) {
	g.clockManager.SetTimeProvider(provider)
}

// SetClockSyncHandler sets the handler for S2F31 time set requests.
// If not set, S2F31 will return TIACK=1 (not allowed).
func (g *GemHandler) SetClockSyncHandler(handler ClockSyncHandler) {
	g.clockManager.SetClockSyncHandler(handler)
}

// GetEquipmentTime returns current equipment time.
func (g *GemHandler) GetEquipmentTime() string {
	return g.clockManager.GetFormattedTime()
}
