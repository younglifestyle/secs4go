package gem

import (
	"fmt"
	"sync"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

// EquipmentConstantValueProvider returns the current value of an equipment constant.
type EquipmentConstantValueProvider func() (ast.ItemNode, error)

// EquipmentConstantValueUpdater applies an updated value received from the remote side.
type EquipmentConstantValueUpdater func(ast.ItemNode) error

// EquipmentConstantValueValidator validates a value prior to being applied.
type EquipmentConstantValueValidator func(ast.ItemNode) error

// EquipmentConstant encapsulates metadata and behaviour for a GEM equipment constant.
type EquipmentConstant struct {
	info idInfo

	Name string
	Unit string

	MinValue     ast.ItemNode
	MaxValue     ast.ItemNode
	DefaultValue ast.ItemNode

	mu        sync.RWMutex
	value     ast.ItemNode
	provider  EquipmentConstantValueProvider
	updater   EquipmentConstantValueUpdater
	validator EquipmentConstantValueValidator
}

// NewEquipmentConstant creates a new equipment constant definition.
//
// id must be a non-negative integer or ASCII string. defaultValue must not be nil.
func NewEquipmentConstant(id interface{}, name string, defaultValue ast.ItemNode, opts ...EquipmentConstantOption) (*EquipmentConstant, error) {
	if defaultValue == nil {
		return nil, fmt.Errorf("equipment constant default value is required")
	}

	idInfo, err := newIDInfo(id)
	if err != nil {
		return nil, err
	}

	ec := &EquipmentConstant{
		info:         idInfo,
		Name:         name,
		DefaultValue: defaultValue,
		value:        defaultValue,
	}

	for _, opt := range opts {
		opt(ec)
	}

	return ec, nil
}

// EquipmentConstantOption mutates an EquipmentConstant at construction time.
type EquipmentConstantOption func(*EquipmentConstant)

// WithEquipmentConstantUnit sets the engineering unit string.
func WithEquipmentConstantUnit(unit string) EquipmentConstantOption {
	return func(ec *EquipmentConstant) {
		ec.Unit = unit
	}
}

// WithEquipmentConstantMin registers the minimum permitted value (for informational purposes).
func WithEquipmentConstantMin(min ast.ItemNode) EquipmentConstantOption {
	return func(ec *EquipmentConstant) {
		ec.MinValue = min
	}
}

// WithEquipmentConstantMax registers the maximum permitted value (for informational purposes).
func WithEquipmentConstantMax(max ast.ItemNode) EquipmentConstantOption {
	return func(ec *EquipmentConstant) {
		ec.MaxValue = max
	}
}

// WithEquipmentConstantValueProvider installs a dynamic value callback.
func WithEquipmentConstantValueProvider(provider EquipmentConstantValueProvider) EquipmentConstantOption {
	return func(ec *EquipmentConstant) {
		ec.provider = provider
	}
}

// WithEquipmentConstantValueUpdater installs a handler invoked when the host updates the constant.
func WithEquipmentConstantValueUpdater(updater EquipmentConstantValueUpdater) EquipmentConstantOption {
	return func(ec *EquipmentConstant) {
		ec.updater = updater
	}
}

// WithEquipmentConstantValidator registers custom validation invoked prior to updates.
func WithEquipmentConstantValidator(validator EquipmentConstantValueValidator) EquipmentConstantOption {
	return func(ec *EquipmentConstant) {
		ec.validator = validator
	}
}

// ID returns the identifier used on the wire (uint64 or string).
func (ec *EquipmentConstant) ID() interface{} {
	return ec.info.raw
}

func (ec *EquipmentConstant) idKey() string {
	return ec.info.key
}

func (ec *EquipmentConstant) idNode() ast.ItemNode {
	return ec.info.node
}

// Value resolves the current value, preferring the provider if present.
func (ec *EquipmentConstant) Value() (ast.ItemNode, error) {
	ec.mu.RLock()
	provider := ec.provider
	value := ec.value
	ec.mu.RUnlock()

	if provider != nil {
		return provider()
	}
	if value != nil {
		return value, nil
	}
	return ec.DefaultValue, nil
}

// ApplyValue stores or forwards a new value.
func (ec *EquipmentConstant) ApplyValue(node ast.ItemNode) error {
	if node == nil {
		return fmt.Errorf("nil value provided for equipment constant %v", ec.ID())
	}

	ec.mu.RLock()
	validator := ec.validator
	updater := ec.updater
	provider := ec.provider
	ec.mu.RUnlock()

	if validator != nil {
		if err := validator(node); err != nil {
			return err
		}
	}

	if updater != nil {
		if err := updater(node); err != nil {
			return err
		}
	}

	if provider == nil {
		ec.mu.Lock()
		ec.value = node
		ec.mu.Unlock()
	}

	return nil
}

// EquipmentConstantValue represents an ECID and its associated value payload.
type EquipmentConstantValue struct {
	ID    interface{}
	Value ast.ItemNode
}

// EquipmentConstantInfo returns metadata supplied by S2F30.
type EquipmentConstantInfo struct {
	ID      interface{}
	Name    string
	Unit    string
	Min     ast.ItemNode
	Max     ast.ItemNode
	Default ast.ItemNode
}

// EquipmentConstantUpdate bundles an outbound update destined for S2F15.
type EquipmentConstantUpdate struct {
	ID    interface{}
	Value ast.ItemNode
}
