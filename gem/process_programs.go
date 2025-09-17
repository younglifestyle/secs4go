package gem

import (
	"fmt"
	"sync"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// ProcessProgram encapsulates a SEMI E30 process program (recipe) identified by PPID.
type ProcessProgram struct {
	info idInfo
	Body string
}

// ProcessProgramUploadHandler allows applications to validate or persist uploaded process programs.
// Returning a non-zero value propagates as the PPACK acknowledgement code.
type ProcessProgramUploadHandler func(ppid interface{}, body string) int

// ProcessProgramRequestHandler is invoked when the host requests a process program via S7F5.
// The handler returns the program body and the PPACK acknowledgement code.
type ProcessProgramRequestHandler func(ppid interface{}) (body string, ack int)

func newProcessProgram(id interface{}, body string) (*ProcessProgram, error) {
	info, err := newIDInfo(id)
	if err != nil {
		return nil, err
	}
	return &ProcessProgram{info: info, Body: body}, nil
}

func (pp *ProcessProgram) ID() interface{} {
	return pp.info.raw
}

func (pp *ProcessProgram) idKey() string {
	return pp.info.key
}

func (pp *ProcessProgram) idNode() ast.ItemNode {
	return pp.info.node
}

type processProgramStore struct {
	mu    sync.RWMutex
	items map[string]*ProcessProgram
}

func newProcessProgramStore() *processProgramStore {
	return &processProgramStore{items: make(map[string]*ProcessProgram)}
}

func (s *processProgramStore) put(pp *ProcessProgram) {
	s.mu.Lock()
	s.items[pp.idKey()] = pp
	s.mu.Unlock()
}

func (s *processProgramStore) get(key string) (*ProcessProgram, bool) {
	s.mu.RLock()
	pp, ok := s.items[key]
	s.mu.RUnlock()
	return pp, ok
}

func (s *processProgramStore) list() []*ProcessProgram {
	s.mu.RLock()
	result := make([]*ProcessProgram, 0, len(s.items))
	for _, pp := range s.items {
		result = append(result, pp)
	}
	s.mu.RUnlock()
	return result
}

func (s *processProgramStore) delete(key string) {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

func ensureProcessProgramKey(id interface{}) (string, error) {
	info, err := newIDInfo(id)
	if err != nil {
		return "", fmt.Errorf("invalid PPID %v: %w", id, err)
	}
	return info.key, nil
}
