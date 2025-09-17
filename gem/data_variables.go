package gem

import (
	"fmt"
	"sync"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// DataValueProvider returns the current value for a data variable.
type DataValueProvider func() (ast.ItemNode, error)

// DataVariable represents a GEM data value (VID) definition.
type DataVariable struct {
	info idInfo

	Name string

	mu       sync.RWMutex
	value    ast.ItemNode
	provider DataValueProvider
}

// DataVariableOption mutates a DataVariable at construction time.
type DataVariableOption func(*DataVariable)

// NewDataVariable constructs a data variable definition. id must be a non-negative integer or ASCII string.
func NewDataVariable(id interface{}, name string, opts ...DataVariableOption) (*DataVariable, error) {
	info, err := newIDInfo(id)
	if err != nil {
		return nil, err
	}

	dv := &DataVariable{
		info: info,
		Name: name,
	}

	for _, opt := range opts {
		opt(dv)
	}

	return dv, nil
}

// WithDataValue sets a static value for the data variable.
func WithDataValue(node ast.ItemNode) DataVariableOption {
	return func(dv *DataVariable) {
		dv.value = node
	}
}

// WithDataValueProvider installs a dynamic provider callback for the data variable.
func WithDataValueProvider(provider DataValueProvider) DataVariableOption {
	return func(dv *DataVariable) {
		dv.provider = provider
	}
}

// ID returns the identifier (uint64 or string).
func (dv *DataVariable) ID() interface{} {
	return dv.info.raw
}

func (dv *DataVariable) idKey() string {
	return dv.info.key
}

func (dv *DataVariable) idNode() ast.ItemNode {
	return dv.info.node
}

// SetValue updates the stored static value. Ignored when a provider is present.
func (dv *DataVariable) SetValue(node ast.ItemNode) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.value = node
}

// SetValueProvider installs or replaces the dynamic value provider.
func (dv *DataVariable) SetValueProvider(provider DataValueProvider) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.provider = provider
}

// Value resolves the current value using provider first, falling back to stored value.
func (dv *DataVariable) Value() (ast.ItemNode, error) {
	dv.mu.RLock()
	provider := dv.provider
	value := dv.value
	dv.mu.RUnlock()

	if provider != nil {
		return provider()
	}
	if value == nil {
		return nil, fmt.Errorf("data variable %v has no value", dv.ID())
	}
	return value, nil
}
