package gem

import (
	"io"
	"strings"
	"time"

	"github.com/younglifestyle/secs4go/hsms"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options describes configurable parameters when creating a GEM handler.
type Options struct {
	Protocol                   *hsms.HsmsProtocol
	DeviceType                 DeviceType
	DeviceID                   uint16
	MDLN                       string
	SOFTREV                    string
	EstablishCommunicationWait time.Duration
	InitialControlState        ControlState
	InitialOnlineMode          OnlineControlMode
	Logging                    LoggingOptions
	Logger                     Logger // Optional: custom structured logger. Defaults to NopLogger().
}

// LoggingOptions configures HSMS/GEM message logging.
type LoggingOptions struct {
	Enabled                    bool
	Mode                       hsms.LoggingMode
	IncludeControlMessages     bool
	ExcludeControlMessageTypes []string
	Writer                     io.Writer

	// Log rotation options
	LogFile    string // Path to log file. If empty, Writer or default is used.
	MaxSize    int    // Max size in megabytes before rotation. Default 100MB.
	MaxBackups int    // Max number of old log files to retain.
	MaxAge     int    // Max number of days to retain old log files.
	Compress   bool   // Compress old log files (gzip).
}

func (o *LoggingOptions) applyDefaults() {
	if !o.Enabled {
		return
	}
	if o.Mode == hsms.LoggingModeUnset {
		o.Mode = hsms.LoggingModeSML
	}
}

func (o LoggingOptions) toConfig(defaultWriter io.Writer) hsms.LoggingConfig {
	writer := o.Writer
	if o.LogFile != "" {
		writer = &lumberjack.Logger{
			Filename:   o.LogFile,
			MaxSize:    o.MaxSize, // megabytes
			MaxBackups: o.MaxBackups,
			MaxAge:     o.MaxAge, // days
			Compress:   o.Compress,
		}
	} else if writer == nil {
		writer = defaultWriter
	}

	cfg := hsms.LoggingConfig{
		Enabled:                     o.Enabled,
		Mode:                        o.Mode,
		IncludeControlMessages:      o.IncludeControlMessages,
		ExcludedControlMessageTypes: make(map[string]struct{}),
		Writer:                      writer,
	}
	for _, msgType := range o.ExcludeControlMessageTypes {
		typeKey := strings.ToLower(strings.TrimSpace(msgType))
		if typeKey == "" {
			continue
		}
		cfg.ExcludedControlMessageTypes[typeKey] = struct{}{}
	}
	if len(cfg.ExcludedControlMessageTypes) == 0 {
		cfg.ExcludedControlMessageTypes = nil
	}
	if cfg.Writer == nil {
		cfg.Writer = defaultWriter
	}
	return cfg
}

func (o *Options) applyDefaults() {
	o.Logging.applyDefaults()
	if o.MDLN == "" {
		if o.DeviceType == DeviceEquipment {
			o.MDLN = "secs4go"
		} else {
			o.MDLN = "host"
		}
	}
	if o.SOFTREV == "" {
		o.SOFTREV = "0.1.0"
	}
	if o.EstablishCommunicationWait == 0 {
		o.EstablishCommunicationWait = 10 * time.Second
	}
	if !isValidInitialControlState(o.InitialControlState) {
		o.InitialControlState = ControlStateAttemptOnline
	}
	if !isValidOnlineMode(o.InitialOnlineMode) {
		o.InitialOnlineMode = OnlineModeRemote
	}
}
