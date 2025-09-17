package hsms

import (
	"context"
	"fmt"
	"github.com/looplab/fsm"
)

var (
	StateNotConnected         = "NOT-CONNECTED"
	StateConnected            = "CONNECTED" // Contains the following two states
	StateConnectedNotSelected = StateConnected + "_" + "NOT-SELECTED"
	StateConnectedSelected    = StateConnected + "_" + "SELECTED"
)

type ConnectionStateMachine struct {
	fsm *fsm.FSM
}

// NewConnectionStateMachine Callbacks : enter_CONNECTED、leave_CONNECTED、enter_CONNECTED_SELECTED
func NewConnectionStateMachine(callbacks fsm.Callbacks) *ConnectionStateMachine {

	cs := &ConnectionStateMachine{}

	if len(callbacks) == 0 {
		callbacks = fsm.Callbacks{
			"leave_" + StateNotConnected:      cs.onEnterConnected,
			"enter_" + StateNotConnected:      cs.onExitConnected,
			"enter_" + StateConnectedSelected: cs.onEnterConnectedSelected,
		}
	}

	cs.fsm = fsm.NewFSM(
		StateNotConnected,
		fsm.Events{
			{Name: "connect", Src: []string{StateNotConnected}, Dst: StateConnectedNotSelected},
			{Name: "disconnect", Src: []string{StateConnectedNotSelected, StateConnectedSelected}, Dst: StateNotConnected},
			{Name: "select", Src: []string{StateConnectedNotSelected}, Dst: StateConnectedSelected},
			{Name: "deselect", Src: []string{StateConnectedSelected}, Dst: StateConnectedNotSelected},
			{Name: "timeoutT7", Src: []string{StateConnectedNotSelected}, Dst: StateNotConnected},
		},
		callbacks,
	)

	return cs
}

func (cs *ConnectionStateMachine) CurrentState() string {
	return cs.fsm.Current()
}

func (cs *ConnectionStateMachine) onEnterConnected(ctx context.Context, e *fsm.Event) {
	fmt.Println("_on_enter_connected")
}

func (cs *ConnectionStateMachine) onExitConnected(ctx context.Context, e *fsm.Event) {
	fmt.Println("_on_exit_connected")
}

func (cs *ConnectionStateMachine) onEnterConnectedSelected(ctx context.Context, e *fsm.Event) {
	fmt.Println("_on_enter_connected_selected")
}

func (cs *ConnectionStateMachine) Connect() error {
	return cs.fsm.Event(context.Background(), "connect")
}

func (cs *ConnectionStateMachine) Disconnect() error {
	return cs.fsm.Event(context.Background(), "disconnect")
}

func (cs *ConnectionStateMachine) Select() error {
	return cs.fsm.Event(context.Background(), "select")
}

func (cs *ConnectionStateMachine) Deselect() error {
	return cs.fsm.Event(context.Background(), "deselect")
}

func (cs *ConnectionStateMachine) TimeoutT7() error {
	return cs.fsm.Event(context.Background(), "timeoutT7")
}
