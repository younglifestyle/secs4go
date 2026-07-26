package utils

import (
	"errors"
	"sync"
	"testing"
)

func TestDequeWakesAllWaiters(t *testing.T) {
	queue := NewDeque()
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := queue.Get(1)
			if err != nil {
				t.Errorf("Get returned error: %v", err)
				return
			}
			results <- value.(int)
		}()
	}
	queue.Put(1)
	queue.Put(2)
	wg.Wait()
	close(results)
	seen := map[int]bool{}
	for result := range results {
		seen[result] = true
	}
	if !seen[1] || !seen[2] {
		t.Fatalf("unexpected results: %v", seen)
	}
}

func TestDequeCloseWakesWaiter(t *testing.T) {
	queue := NewDeque()
	want := errors.New("connection closed")
	result := make(chan error, 1)
	go func() {
		_, err := queue.Get(10)
		result <- err
	}()
	queue.Close(want)
	if got := <-result; !errors.Is(got, want) {
		t.Fatalf("Get error = %v, want %v", got, want)
	}
}
