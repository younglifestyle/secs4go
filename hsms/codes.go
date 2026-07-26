package hsms

// ControlStatus encodes HSMS select/deselect response statuses.
type ControlStatus byte

const (
	ControlStatusAccepted ControlStatus = 0
	ControlStatusDenied   ControlStatus = 1
)

// RejectReason enumerates HSMS reject codes used by the stack.
type RejectReason byte

const (
	RejectReasonSTypeNotSupported  RejectReason = 1
	RejectReasonPTypeNotSupported  RejectReason = 2
	RejectReasonTransactionNotOpen RejectReason = 3
	RejectReasonNotSelected        RejectReason = 4

	// Deprecated aliases retained for source compatibility.
	RejectReasonBusyOrAlreadyActive = RejectReasonPTypeNotSupported
	RejectReasonNotReady            = RejectReasonNotSelected
)
