package gem

import "github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"

// ReportValue captures the values associated with a report identifier in an S6F11/S6F16 payload.
type ReportValue struct {
	RPTID  interface{}
	Values []ast.ItemNode
}

// EventReport represents a decoded collection event report message.
type EventReport struct {
	DATAID  int
	CEID    interface{}
	Reports []ReportValue
}
