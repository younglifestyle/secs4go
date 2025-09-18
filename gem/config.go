package gem

import (
	"io"
	"time"

	"github.com/younglifestyle/secs4go/hsms"
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
}

// LoggingOptions configures HSMS/GEM message logging.
type LoggingOptions struct {
	Enabled                bool
	Mode                   hsms.LoggingMode
	IncludeControlMessages bool
	Writer                 io.Writer
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
	cfg := hsms.LoggingConfig{
		Enabled:                o.Enabled,
		Mode:                   o.Mode,
		IncludeControlMessages: o.IncludeControlMessages,
		Writer:                 o.Writer,
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
