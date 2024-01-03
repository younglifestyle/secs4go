package common

import (
	"sync"
)

// Event represents a class to handle the callbacks for a single event.
type Event struct {
	callbacks []func(data map[string]interface{})
	mutex     sync.Mutex
}

// AddCallback adds a new callback to the event.
func (e *Event) AddCallback(callback func(data map[string]interface{})) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.callbacks = append(e.callbacks, callback)
}

// RemoveCallback removes a callback from the event.
func (e *Event) RemoveCallback(callback func(data map[string]interface{})) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i, cb := range e.callbacks {
		if &cb == &callback {
			// Remove the callback by copying the slice without the element at index i.
			e.callbacks = append(e.callbacks[:i], e.callbacks[i+1:]...)
			return
		}
	}
}

// Fire raises the event and calls all callbacks.
func (e *Event) Fire(data map[string]interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, callback := range e.callbacks {
		callback(data)
	}
}

// Len returns the number of callbacks.
func (e *Event) Len() int {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return len(e.callbacks)
}

//func main() {
//	// Create an instance of Event.
//	event := &Event{}
//
//	// Add some callbacks to the event.
//	callback1 := func(data map[string]interface{}) {
//		fmt.Println("Callback 1 executed with data:", data)
//	}
//	callback2 := func(data map[string]interface{}) {
//		fmt.Println("Callback 2 executed with data:", data)
//	}
//	event.AddCallback(callback1)
//	event.AddCallback(callback2)
//
//	// Fire the event with some data.
//	data := map[string]interface{}{"key": "value"}
//	event.Fire(data)
//
//	// Remove a callback from the event.
//	event.RemoveCallback(callback1)
//
//	// Fire the event again with updated data.
//	data = map[string]interface{}{"another_key": "another_value"}
//	event.Fire(data)
//
//	// Get the number of callbacks in the event.
//	numCallbacks := event.Len()
//	fmt.Println("Number of callbacks:", numCallbacks)
//}
