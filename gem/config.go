package gem

import (
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
}

func (o *Options) applyDefaults() {
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
