package utils

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

type Deque struct {
	sync.RWMutex
	notEmptyNotify chan struct{}
	container      *list.List
	closed         bool
	closeErr       error
}

func NewDeque() *Deque {
	return &Deque{container: list.New(), notEmptyNotify: make(chan struct{})}
}

func (s *Deque) Put(item interface{}) {
	s.Lock()
	if s.closed {
		s.Unlock()
		return
	}
	s.container.PushFront(item)
	close(s.notEmptyNotify)
	s.notEmptyNotify = make(chan struct{})
	s.Unlock()
}

// Close wakes pending getters and makes future Get calls return err. Items
// already queued remain readable before the close error is reported.
func (s *Deque) Close(err error) {
	if err == nil {
		err = errors.New("deque closed")
	}
	s.Lock()
	if !s.closed {
		s.closed = true
		s.closeErr = err
		close(s.notEmptyNotify)
	}
	s.Unlock()
}

func (s *Deque) Get(timeout int) (interface{}, error) {
	endTime := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		s.Lock()
		if last := s.container.Back(); last != nil {
			item := s.container.Remove(last)
			s.Unlock()
			return item, nil
		}
		if s.closed {
			err := s.closeErr
			s.Unlock()
			return nil, err
		}
		notify := s.notEmptyNotify
		s.Unlock()

		remaining := endTime.Sub(time.Now())
		if remaining <= 0 {
			return nil, errors.New("time out in Get")
		}
		timer := time.NewTimer(remaining)
		select {
		case <-notify:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			return nil, errors.New("time out in Get")
		}
	}
}
