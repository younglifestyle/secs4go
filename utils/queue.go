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
}

func NewDeque() *Deque {
	return &Deque{container: list.New(), notEmptyNotify: make(chan struct{})}
}

func (s *Deque) Put(item interface{}) {
	s.Lock()
	s.container.PushFront(item)
	s.Unlock()
	select {
	case s.notEmptyNotify <- struct{}{}:
	default:
	}
}

func (s *Deque) Get(timeout int) (interface{}, error) {
	s.Lock()
	var item interface{} = nil
	var lastContainerItem *list.Element = nil
	endTime := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		if s.container.Back() != nil {
			break
		}
		remaining := endTime.Sub(time.Now())
		s.Unlock()
		if remaining < 0 {
			return nil, errors.New("time out in Get")
		}
		select {
		case <-s.notEmptyNotify:
		case <-time.After(remaining):
			return nil, errors.New("time out in Get")
		}
		s.Lock()
	}
	lastContainerItem = s.container.Back()
	item = s.container.Remove(lastContainerItem)
	s.Unlock()
	return item, nil
}
