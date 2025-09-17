package gem

import (
	"errors"
	"testing"
)

func TestControlStateMachineDefaultStart(t *testing.T) {
	sm := NewControlStateMachine(ControlStateMachineOptions{})

	requireControlState(t, sm, ControlStateAttemptOnline)
	if mode := sm.OnlineMode(); mode != OnlineModeRemote {
		t.Fatalf("expected default online mode %s, got %s", OnlineModeRemote, mode)
	}

	if err := sm.AttemptOnlineFailHostOffline(); err != nil {
		t.Fatalf("attempt_online_fail_host_offline: %v", err)
	}
	requireControlState(t, sm, ControlStateHostOffline)
}

func TestControlStateMachineOnlineTransitions(t *testing.T) {
	sm := NewControlStateMachine(ControlStateMachineOptions{InitialState: ControlStateEquipmentOffline})

	requireControlState(t, sm, ControlStateEquipmentOffline)

	if err := sm.SwitchOnline(); err != nil {
		t.Fatalf("switch_online: %v", err)
	}
	requireControlState(t, sm, ControlStateAttemptOnline)

	if err := sm.AttemptOnlineSuccess(); err != nil {
		t.Fatalf("attempt_online_success: %v", err)
	}
	requireControlState(t, sm, ControlStateOnlineRemote)

	if err := sm.SwitchOnlineLocal(); err != nil {
		t.Fatalf("switch_online_local: %v", err)
	}
	requireControlState(t, sm, ControlStateOnlineLocal)

	if err := sm.RemoteOffline(); err != nil {
		t.Fatalf("remote_offline: %v", err)
	}
	requireControlState(t, sm, ControlStateHostOffline)

	if err := sm.RemoteOnline(); err != nil {
		t.Fatalf("remote_online: %v", err)
	}
	requireControlState(t, sm, ControlStateOnlineLocal)

	if err := sm.SwitchOnlineRemote(); err != nil {
		t.Fatalf("switch_online_remote: %v", err)
	}
	requireControlState(t, sm, ControlStateOnlineRemote)

	if err := sm.SwitchOffline(); err != nil {
		t.Fatalf("switch_offline: %v", err)
	}
	requireControlState(t, sm, ControlStateEquipmentOffline)
}

func TestControlStateMachineInvalidTransitions(t *testing.T) {
	sm := NewControlStateMachine(ControlStateMachineOptions{InitialState: ControlStateEquipmentOffline})

	if err := sm.invalidTransitionLocked("test"); err == nil {
		t.Fatal("invalidTransitionLocked returned nil")
	}

	err := sm.SwitchOnlineLocal()
	if err == nil {
		t.Fatalf("expected invalid transition error, got nil (state=%s)", sm.State())
	} else if !errors.Is(err, ErrInvalidControlTransition) {
		t.Fatalf("expected invalid transition error, got %v (state=%s, type=%T)", err, sm.State(), err)
	}

	if err := sm.RemoteOnline(); err == nil {
		t.Fatalf("expected invalid transition error, got nil (state=%s)", sm.State())
	} else if !errors.Is(err, ErrInvalidControlTransition) {
		t.Fatalf("expected invalid transition error, got %v (state=%s)", err, sm.State())
	}

	if err := sm.SwitchOffline(); err == nil {
		t.Fatalf("expected invalid transition error, got nil (state=%s)", sm.State())
	} else if !errors.Is(err, ErrInvalidControlTransition) {
		t.Fatalf("expected invalid transition error, got %v (state=%s)", err, sm.State())
	}
}
func requireControlState(t *testing.T, sm *ControlStateMachine, want ControlState) {
	t.Helper()
	if got := sm.State(); got != want {
		t.Fatalf("unexpected control state: got %s, want %s", got, want)
	}
}
