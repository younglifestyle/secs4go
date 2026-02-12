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
	RejectReasonBusyOrAlreadyActive RejectReason = 2
	RejectReasonNotReady            RejectReason = 4
)
