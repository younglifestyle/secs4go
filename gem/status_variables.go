package gem

import (
	"fmt"
	"sync"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// StatusValueProvider returns the current value for a status variable.
type StatusValueProvider func() (ast.ItemNode, error)

// StatusVariable describes a single GEM status variable configuration.
type StatusVariable struct {
	info idInfo
	Name string
	Unit string

	mu       sync.RWMutex
	value    ast.ItemNode
	provider StatusValueProvider
}

// NewStatusVariable constructs a status variable definition.
//
// id must be a non-negative integer or ASCII string.
func NewStatusVariable(id interface{}, name, unit string, opts ...StatusVariableOption) (*StatusVariable, error) {
	idInfo, err := newIDInfo(id)
	if err != nil {
		return nil, err
	}

	sv := &StatusVariable{
		info: idInfo,
		Name: name,
		Unit: unit,
	}

	for _, opt := range opts {
		opt(sv)
	}

	return sv, nil
}

// StatusVariableOption mutates a StatusVariable during construction.
type StatusVariableOption func(*StatusVariable)

// WithStatusValue sets a static value for the status variable.
func WithStatusValue(node ast.ItemNode) StatusVariableOption {
	return func(sv *StatusVariable) {
		sv.value = node
	}
}

// WithStatusValueProvider registers a dynamic provider callback.
func WithStatusValueProvider(provider StatusValueProvider) StatusVariableOption {
	return func(sv *StatusVariable) {
		sv.provider = provider
	}
}

// ID returns the application facing identifier (uint64 or string).
func (sv *StatusVariable) ID() interface{} {
	return sv.info.raw
}

// idKey returns the internal cache key.
func (sv *StatusVariable) idKey() string {
	return sv.info.key
}

// idNode returns the identifier encoded as an ItemNode.
func (sv *StatusVariable) idNode() ast.ItemNode {
	return sv.info.node
}

// SetValue updates the stored static value. Dynamic providers should use callbacks instead.
func (sv *StatusVariable) SetValue(node ast.ItemNode) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.value = node
}

// SetValueProvider installs or replaces the dynamic value provider.
func (sv *StatusVariable) SetValueProvider(provider StatusValueProvider) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.provider = provider
}

// Value resolves the current value either via provider or stored static value.
func (sv *StatusVariable) Value() (ast.ItemNode, error) {
	sv.mu.RLock()
	provider := sv.provider
	value := sv.value
	sv.mu.RUnlock()

	if provider != nil {
		return provider()
	}
	if value == nil {
		return nil, fmt.Errorf("status variable %v has no value", sv.ID())
	}
	return value, nil
}

// StatusValue couples an identifier with the returned ItemNode payload.
type StatusValue struct {
	ID    interface{}
	Value ast.ItemNode
}

// StatusVariableInfo describes metadata returned by S1F12.
type StatusVariableInfo struct {
	ID   interface{}
	Name string
	Unit string
}
