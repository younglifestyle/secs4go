package ast

import (
	"fmt"
	"strings"
)

// BinaryNode is a immutable data type that represents a binary data item in a SECS-II message.
// Implements ItemNode.
type BooleanNode struct {
	values    []bool         // Array of boolean values
	variables map[string]int // Variable name and its position in the data array

	symbol string
	// Rep invariants
	// - If a variable exists in position i, values[i] will be zero-value (false) and should not be used.
	// - variable name should adhere to the variable naming rule; refer to interface.go
	// - variable positions should be unique, and be in range of [0, len(values))
}

func (node *BooleanNode) Values() interface{} {
	return node.values
}

func (node *BooleanNode) Type() string {
	return node.symbol
}

// Factory methods

// NewBooleanNode creates a new BooleanNode.
//
// Each input argument should be a bool, or a string with a valid variable name
// as specified in the interface documentation.
func NewBooleanNode(values ...interface{}) ItemNode {
	if getDataByteLength("boolean", len(values)) > MAX_BYTE_SIZE {
		panic("item node size limit exceeded")
	}

	var (
		nodeValues    []bool         = make([]bool, 0, len(values))
		nodeVariables map[string]int = make(map[string]int)
	)

	for i, value := range values {
		if v, ok := value.(bool); ok {
			// value is a bool
			nodeValues = append(nodeValues, v)
		} else if v, ok := value.(string); ok {
			// value is a variable
			if _, ok := nodeVariables[v]; ok {
				panic("duplicated variable name found")
			}
			nodeVariables[v] = i
			nodeValues = append(nodeValues, false)
		} else {
			panic("input argument contains invalid type for BooleanNode")
		}
	}

	node := &BooleanNode{nodeValues, nodeVariables, "boolean"}
	node.checkRep()
	return node
}

// Public methods

func (node *BooleanNode) Get(indices ...int) (ItemNode, error) {
	if len(indices) == 0 {
		return node, nil
	} else {
		return nil, fmt.Errorf("not list, node is %s, indices is %v", node, indices)
	}
}

// Size implements ItemNode.Size().
func (node *BooleanNode) Size() int {
	return len(node.values)
}

// Variables implements ItemNode.Variables().
func (node *BooleanNode) Variables() []string {
	return getVariableNames(node.variables)
}

// FillVariables implements ItemNode.FillVariables().
func (node *BooleanNode) FillVariables(values map[string]interface{}) ItemNode {
	if len(node.variables) == 0 {
		return node
	}

	nodeValues := make([]interface{}, 0, node.Size())
	for _, v := range node.values {
		nodeValues = append(nodeValues, v)
	}

	createNew := false
	for name, pos := range node.variables {
		if v, ok := values[name]; ok {
			nodeValues[pos] = v
			createNew = true
		} else {
			nodeValues[pos] = name
		}
	}

	if !createNew {
		return node
	}
	return NewBooleanNode(nodeValues...)
}

// ToBytes implements ItemNode.ToBytes()
func (node *BooleanNode) ToBytes() []byte {
	if len(node.variables) != 0 {
		return []byte{}
	}

	result, err := getHeaderBytes("boolean", node.Size())
	if err != nil {
		return []byte{}
	}

	for _, value := range node.values {
		if value {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
	}

	return result
}

// String returns the string representation of the node.
func (node *BooleanNode) String() string {
	if node.Size() == 0 {
		return "<BOOLEAN[0]>"
	}

	values := make([]string, 0, node.Size())
	for _, value := range node.values {
		if value {
			values = append(values, "T")
		} else {
			values = append(values, "F")
		}
	}

	for name, pos := range node.variables {
		values[pos] = name
	}

	return fmt.Sprintf("<BOOLEAN[%d] %v>", node.Size(), strings.Join(values, " "))
}

// Private methods

func (node *BooleanNode) checkRep() {
	visited := map[int]bool{}
	for name, pos := range node.variables {
		if node.values[pos] {
			panic("value in variable position isn't a zero-value")
		}

		if !isValidVarName(name) {
			panic("invalid variable name")
		}

		if _, ok := visited[pos]; ok {
			panic("variable position is not unique")
		}
		visited[pos] = true

		if !(0 <= pos && pos < node.Size()) {
			panic("variable position overflow")
		}
	}
}
