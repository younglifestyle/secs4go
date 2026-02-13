package gem

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ZapLoggerOptions configures the zap-based file logger.
type ZapLoggerOptions struct {
	// LogFile is the path to the log file.
	// If empty, logs are written to stdout.
	LogFile string

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool

	// DebugLevel enables debug logging if true. Otherwise, Info level is used.
	DebugLevel bool

	// Console, if true, writes logs to stdout IN ADDITION to the log file.
	// If LogFile is empty, this option is ignored (stdout is already used).
	Console bool
}

// NewZapLogger creates a Logger backed by uber-go/zap with optional file rotation.
func NewZapLogger(opts ZapLoggerOptions) Logger {
	// Determine write syncer
	var ws zapcore.WriteSyncer

	if opts.LogFile != "" {
		lj := &lumberjack.Logger{
			Filename:   opts.LogFile,
			MaxSize:    opts.MaxSize,
			MaxBackups: opts.MaxBackups,
			MaxAge:     opts.MaxAge,
			Compress:   opts.Compress,
		}
		if opts.Console {
			ws = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lj))
		} else {
			ws = zapcore.AddSync(lj)
		}
	} else {
		ws = zapcore.AddSync(os.Stdout)
	}

	// Encoder config
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encCfg)

	// Level
	level := zap.InfoLevel
	if opts.DebugLevel {
		level = zap.DebugLevel
	}

	core := zapcore.NewCore(encoder, ws, level)
	logger := zap.New(core)

	return &zapAdapter{s: logger.Sugar()}
}

// zapAdapter adapts zap.SugaredLogger to our Logger interface.
type zapAdapter struct {
	s *zap.SugaredLogger
}

func (z *zapAdapter) Debug(msg string, kv ...interface{}) {
	z.s.Debugw(msg, kv...)
}

func (z *zapAdapter) Info(msg string, kv ...interface{}) {
	z.s.Infow(msg, kv...)
}

func (z *zapAdapter) Warn(msg string, kv ...interface{}) {
	z.s.Warnw(msg, kv...)
}

func (z *zapAdapter) Error(msg string, kv ...interface{}) {
	z.s.Errorw(msg, kv...)
}
