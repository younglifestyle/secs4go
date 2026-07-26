package common

import "testing"

func TestEventRemoveCallback(t *testing.T) {
	event := &Event{}
	called := 0
	callback := func(map[string]interface{}) { called++ }
	event.AddCallback(callback)
	event.RemoveCallback(callback)
	event.Fire(nil)
	if called != 0 {
		t.Fatalf("removed callback invoked %d times", called)
	}
}

func TestEventCallbackCanModifyEvent(t *testing.T) {
	event := &Event{}
	called := 0
	var callback func(map[string]interface{})
	callback = func(map[string]interface{}) {
		called++
		event.RemoveCallback(callback)
	}
	event.AddCallback(callback)
	event.Fire(nil)
	event.Fire(nil)
	if called != 1 {
		t.Fatalf("callback invoked %d times, want 1", called)
	}
}
