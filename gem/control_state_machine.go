package gem

import (
	"errors"
	"fmt"
	"sync"
)

// ControlState enumerates the GEM control state model states.
type ControlState string

const (
	ControlStateInit             ControlState = "INIT"
	ControlStateControl          ControlState = "CONTROL"
	ControlStateOffline          ControlState = "OFFLINE"
	ControlStateEquipmentOffline ControlState = "EQUIPMENT_OFFLINE"
	ControlStateAttemptOnline    ControlState = "ATTEMPT_ONLINE"
	ControlStateHostOffline      ControlState = "HOST_OFFLINE"
	ControlStateOnline           ControlState = "ONLINE"
	ControlStateOnlineLocal      ControlState = "ONLINE_LOCAL"
	ControlStateOnlineRemote     ControlState = "ONLINE_REMOTE"
)

// OnlineControlMode captures the sub-state of the ONLINE state.
type OnlineControlMode string

const (
	OnlineModeLocal  OnlineControlMode = "LOCAL"
	OnlineModeRemote OnlineControlMode = "REMOTE"
)

// ControlStateMachineOptions configures a ControlStateMachine instance.
type ControlStateMachineOptions struct {
	InitialState      ControlState
	InitialOnlineMode OnlineControlMode
}

// ErrInvalidControlTransition signals an invalid transition request.
var ErrInvalidControlTransition = errors.New("gem: invalid control state transition")

// ControlStateMachine implements the SEMI E30 control state model.
type ControlStateMachine struct {
	mu         sync.RWMutex
	state      ControlState
	initial    ControlState
	onlineMode OnlineControlMode
}

// NewControlStateMachine builds a control state machine initialized to the configured starting state.
func NewControlStateMachine(opts ControlStateMachineOptions) *ControlStateMachine {
	sm := &ControlStateMachine{
		state:      ControlStateInit,
		initial:    ControlStateAttemptOnline,
		onlineMode: OnlineModeRemote,
	}

	if isValidInitialControlState(opts.InitialState) {
		sm.initial = opts.InitialState
	}
	if isValidOnlineMode(opts.InitialOnlineMode) {
		sm.onlineMode = opts.InitialOnlineMode
	}

	sm.Start()
	return sm
}

// Start performs the initial transition sequence when the machine is still in INIT.
func (sm *ControlStateMachine) Start() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != ControlStateInit {
		return
	}

	sm.state = ControlStateControl
	sm.applyInitialLocked()
}

// State returns the current control state.
func (sm *ControlStateMachine) State() ControlState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// OnlineMode reports the currently selected ONLINE sub-state preference.
func (sm *ControlStateMachine) OnlineMode() OnlineControlMode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.onlineMode
}

// SwitchOnline models the operator transition from EQUIPMENT_OFFLINE to ATTEMPT_ONLINE.
func (sm *ControlStateMachine) SwitchOnline() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == ControlStateAttemptOnline {
		return nil
	}
	if sm.state != ControlStateEquipmentOffline {
		return sm.invalidTransitionLocked("switch_online")
	}
	sm.state = ControlStateAttemptOnline
	return nil
}

// SwitchOffline models the operator returning to EQUIPMENT_OFFLINE.
func (sm *ControlStateMachine) SwitchOffline() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isOnlineStateLocked() {
		return sm.invalidTransitionLocked("switch_offline")
	}
	sm.state = ControlStateEquipmentOffline
	return nil
}

// SwitchOnlineLocal models switching from ONLINE_REMOTE to ONLINE_LOCAL.
func (sm *ControlStateMachine) SwitchOnlineLocal() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == ControlStateOnlineLocal {
		sm.onlineMode = OnlineModeLocal
		return nil
	}
	sm.ensureOnlineSubstateLocked()
	if sm.state != ControlStateOnlineRemote {
		return sm.invalidTransitionLocked("switch_online_local")
	}
	sm.onlineMode = OnlineModeLocal
	sm.state = ControlStateOnlineLocal
	return nil
}

// SwitchOnlineRemote models switching from ONLINE_LOCAL to ONLINE_REMOTE.
func (sm *ControlStateMachine) SwitchOnlineRemote() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == ControlStateOnlineRemote {
		sm.onlineMode = OnlineModeRemote
		return nil
	}
	sm.ensureOnlineSubstateLocked()
	if sm.state != ControlStateOnlineLocal {
		return sm.invalidTransitionLocked("switch_online_remote")
	}
	sm.onlineMode = OnlineModeRemote
	sm.state = ControlStateOnlineRemote
	return nil
}

// AttemptOnlineFailEquipmentOffline transitions ATTEMPT_ONLINE to EQUIPMENT_OFFLINE.
func (sm *ControlStateMachine) AttemptOnlineFailEquipmentOffline() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == ControlStateEquipmentOffline {
		return nil
	}
	if sm.state != ControlStateAttemptOnline {
		return sm.invalidTransitionLocked("attempt_online_fail_equipment_offline")
	}
	sm.state = ControlStateEquipmentOffline
	return nil
}

// AttemptOnlineFailHostOffline transitions ATTEMPT_ONLINE to HOST_OFFLINE.
func (sm *ControlStateMachine) AttemptOnlineFailHostOffline() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == ControlStateHostOffline {
		return nil
	}
	if sm.state != ControlStateAttemptOnline {
		return sm.invalidTransitionLocked("attempt_online_fail_host_offline")
	}
	sm.state = ControlStateHostOffline
	return nil
}

// AttemptOnlineSuccess transitions ATTEMPT_ONLINE to ONLINE_(LOCAL|REMOTE).
func (sm *ControlStateMachine) AttemptOnlineSuccess() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isOnlineStateLocked() {
		return nil
	}
	if sm.state != ControlStateAttemptOnline {
		return sm.invalidTransitionLocked("attempt_online_success")
	}
	sm.enterOnlineLocked()
	return nil
}

// RemoteOffline transitions the ONLINE cluster to HOST_OFFLINE.
func (sm *ControlStateMachine) RemoteOffline() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isOnlineStateLocked() {
		return sm.invalidTransitionLocked("remote_offline")
	}
	sm.state = ControlStateHostOffline
	return nil
}

// RemoteOnline transitions HOST_OFFLINE back to ONLINE_(LOCAL|REMOTE).
func (sm *ControlStateMachine) RemoteOnline() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state != ControlStateHostOffline {
		return sm.invalidTransitionLocked("remote_online")
	}
	sm.enterOnlineLocked()
	return nil
}

func (sm *ControlStateMachine) applyInitialLocked() {
	switch sm.initial {
	case ControlStateOnline:
		sm.enterOnlineLocked()
	case ControlStateEquipmentOffline:
		sm.state = ControlStateOffline
		sm.state = ControlStateEquipmentOffline
	case ControlStateHostOffline:
		sm.state = ControlStateOffline
		sm.state = ControlStateHostOffline
	case ControlStateAttemptOnline:
		fallthrough
	default:
		sm.state = ControlStateOffline
		sm.state = ControlStateAttemptOnline
	}
}

func (sm *ControlStateMachine) enterOnlineLocked() {
	sm.state = ControlStateOnline
	sm.applyOnlineModeLocked()
}

func (sm *ControlStateMachine) applyOnlineModeLocked() {
	if !isValidOnlineMode(sm.onlineMode) {
		sm.onlineMode = OnlineModeRemote
	}
	if sm.onlineMode == OnlineModeLocal {
		sm.state = ControlStateOnlineLocal
	} else {
		sm.state = ControlStateOnlineRemote
	}
}

func (sm *ControlStateMachine) ensureOnlineSubstateLocked() {
	if sm.state == ControlStateOnline {
		sm.applyOnlineModeLocked()
	}
}

func (sm *ControlStateMachine) isOnlineStateLocked() bool {
	switch sm.state {
	case ControlStateOnline, ControlStateOnlineLocal, ControlStateOnlineRemote:
		return true
	default:
		return false
	}
}

func (sm *ControlStateMachine) invalidTransitionLocked(action string) error {
	return fmt.Errorf("gem: cannot %s while in control state %s: %w", action, sm.state, ErrInvalidControlTransition)
}

func isValidInitialControlState(state ControlState) bool {
	switch state {
	case ControlStateEquipmentOffline, ControlStateAttemptOnline, ControlStateHostOffline, ControlStateOnline:
		return true
	default:
		return false
	}
}

func isValidOnlineMode(mode OnlineControlMode) bool {
	switch mode {
	case OnlineModeLocal, OnlineModeRemote:
		return true
	default:
		return false
	}
}
