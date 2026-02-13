package gem

import "github.com/younglifestyle/secs4go/common"

// Logger is the logging interface used by secs4go.
// Re-exported from common for user convenience.
type Logger = common.Logger

// NopLogger returns a silent Logger (default).
var NopLogger = common.NopLogger

// NewStdLogger creates a Logger backed by Go's standard log package.
var NewStdLogger = common.NewStdLogger
