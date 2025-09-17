package gem

import (
	"sync"
	"time"
)

// CommunicationState reflects GEM communication status.
type CommunicationState int

const (
	CommunicationStateDisabled CommunicationState = iota
	CommunicationStateNotCommunicating
	CommunicationStateWaitCRA
	CommunicationStateWaitDelay
	CommunicationStateCommunicating
)

type stateMachine struct {
	mu             sync.Mutex
	state          CommunicationState
	waitCRATimer   *time.Timer
	waitDelayTimer *time.Timer
}

func newStateMachine() *stateMachine {
	return &stateMachine{state: CommunicationStateDisabled}
}

func (sm *stateMachine) State() CommunicationState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

func (sm *stateMachine) setState(state CommunicationState) (prev CommunicationState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	prev = sm.state
	if prev == state {
		return prev
	}
	sm.state = state
	sm.stopTimersLocked()
	return prev
}

func (sm *stateMachine) setStateWithWaitCRA(state CommunicationState, duration time.Duration, timeout func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state == state {
		sm.stopWaitCRALocked()
	} else {
		sm.state = state
		sm.stopTimersLocked()
	}
	sm.waitCRATimer = time.AfterFunc(duration, timeout)
}

func (sm *stateMachine) setStateWithWaitDelay(state CommunicationState, duration time.Duration, timeout func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state == state {
		sm.stopWaitDelayLocked()
	} else {
		sm.state = state
		sm.stopTimersLocked()
	}
	sm.waitDelayTimer = time.AfterFunc(duration, timeout)
}

func (sm *stateMachine) stopTimers() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stopTimersLocked()
}

func (sm *stateMachine) stopTimersLocked() {
	sm.stopWaitCRALocked()
	sm.stopWaitDelayLocked()
}

func (sm *stateMachine) stopWaitCRALocked() {
	if sm.waitCRATimer != nil {
		sm.waitCRATimer.Stop()
		sm.waitCRATimer = nil
	}
}

func (sm *stateMachine) stopWaitDelayLocked() {
	if sm.waitDelayTimer != nil {
		sm.waitDelayTimer.Stop()
		sm.waitDelayTimer = nil
	}
}
