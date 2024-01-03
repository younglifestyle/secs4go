package hsms

import (
	"context"
	"fmt"
	. "github.com/ahmetb/go-linq/v3"
	"github.com/looplab/fsm"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Door struct {
	To  string
	FSM *fsm.FSM
}

func NewDoor(to string) *Door {
	d := &Door{
		To: to,
	}

	d.FSM = fsm.NewFSM(
		"closed",
		fsm.Events{
			{Name: "open", Src: []string{"closed"}, Dst: "open"},
			{Name: "close", Src: []string{"open"}, Dst: "closed"},
		},
		fsm.Callbacks{
			"enter_state": func(_ context.Context, e *fsm.Event) { d.enterState(e) },
		},
	)

	return d
}

func (d *Door) enterState(e *fsm.Event) {
	fmt.Printf("The door to %s is %s\n", d.To, e.Dst)
}

func TestFSM2(t *testing.T) {
	door := NewDoor("heaven")

	err := door.FSM.Event(context.Background(), "open")
	if err != nil {
		fmt.Println(err)
	}

	err = door.FSM.Event(context.Background(), "close")
	if err != nil {
		fmt.Println(err)
	}

	err = door.FSM.Event(context.Background(), "close1")
	if err != nil {
		fmt.Println("error : ", err)
	}
}

func TestFSM(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)

	fmt.Println(stateMachine.Connect())
	fmt.Println(stateMachine.CurrentState())

	fmt.Println(stateMachine.Select())
	fmt.Println(stateMachine.CurrentState())
}

func testInitialState(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	assert.Equal(t, stateMachine.CurrentState(), StateNotConnected)
}

func testNotConnected2Connected(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	assert.Equal(t, stateMachine.CurrentState(), StateConnectedNotSelected)
}

func testConnectedNotSelected2NotConnected(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	stateMachine.Disconnect()
	assert.Equal(t, StateNotConnected, stateMachine.CurrentState())
}

func testConnectedSelected2NotConnected(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	stateMachine.Select()
	stateMachine.Disconnect()
	assert.Equal(t, StateNotConnected, stateMachine.CurrentState())
}

func testConnectedNotSelected2ConnectedSelected(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	stateMachine.Select()
	assert.Equal(t, StateConnectedSelected, stateMachine.CurrentState())
}

func testConnectedSelected2ConnectedNotSelected(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	stateMachine.Select()
	stateMachine.Deselect()
	assert.Equal(t, StateConnectedNotSelected, stateMachine.CurrentState())
}

func testConnectedNotSelected2NotConnectedT7(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	stateMachine.TimeoutT7()
	assert.Equal(t, StateNotConnected, stateMachine.CurrentState())
}

func testConnectedConnected(t *testing.T) {
	stateMachine := NewConnectionStateMachine(nil)
	stateMachine.Connect()
	stateMachine.Connect()
	assert.Equal(t, StateConnectedNotSelected, stateMachine.CurrentState())
}

func TestConnectionStateMachine(t *testing.T) {

	t.Run("A=1", func(t *testing.T) {
		TestFSM(t)
	})

	t.Run("testInitialState", func(t *testing.T) {
		testInitialState(t)
	})

	t.Run("testNotConnected2Connected", func(t *testing.T) {
		testNotConnected2Connected(t)
	})

	t.Run("testConnectedNotSelected2NotConnected", func(t *testing.T) {
		testConnectedNotSelected2NotConnected(t)
	})

	t.Run("testConnectedSelected2NotConnected", func(t *testing.T) {
		testConnectedSelected2NotConnected(t)
	})

	t.Run("testConnectedNotSelected2ConnectedSelected", func(t *testing.T) {
		testConnectedNotSelected2ConnectedSelected(t)
	})

	t.Run("testConnectedSelected2ConnectedNotSelected", func(t *testing.T) {
		testConnectedSelected2ConnectedNotSelected(t)
	})

	t.Run("testConnectedNotSelected2NotConnectedT7", func(t *testing.T) {
		testConnectedNotSelected2NotConnectedT7(t)
	})

	t.Run("testConnectedNotSelected2NotConnectedT7", func(t *testing.T) {
		testConnectedNotSelected2NotConnectedT7(t)
	})

	t.Run("testConnectedConnected", func(t *testing.T) {
		testConnectedConnected(t)
	})
}

func TestF1(t *testing.T) {
	type Book struct {
		id      int
		title   string
		authors []string
	}

	books := []Book{
		{
			id:      1,
			title:   "test",
			authors: []string{"12", "34"},
		},
		{
			id:      2,
			title:   "test1",
			authors: []string{"12", "34"},
		},
	}
	author := From(books).SelectMany( // make a flat array of authors
		func(book interface{}) Query {
			return From(book.(Book).authors)
		}).GroupBy( // group by author
		func(author interface{}) interface{} {
			return author // author as key
		}, func(author interface{}) interface{} {
			return author // author as value
		}).OrderByDescending( // sort groups by its length
		func(group interface{}) interface{} {
			return len(group.(Group).Group)
		}).Select( // get authors out of groups
		func(group interface{}) interface{} {
			return group.(Group).Key
		}).First() // take the first author

	fmt.Println(author)
}
