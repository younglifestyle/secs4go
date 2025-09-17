package gem

import (
	"fmt"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

type idInfo struct {
	raw  interface{}
	node ast.ItemNode
	key  string
}

func newIDInfo(value interface{}) (idInfo, error) {
	switch v := value.(type) {
	case uint8:
		return newIDInfoFromUint(uint64(v))
	case uint16:
		return newIDInfoFromUint(uint64(v))
	case uint32:
		return newIDInfoFromUint(uint64(v))
	case uint64:
		return newIDInfoFromUint(v)
	case uint:
		return newIDInfoFromUint(uint64(v))
	case int8:
		if v < 0 {
			return idInfo{}, fmt.Errorf("negative identifier %d not supported", v)
		}
		return newIDInfoFromUint(uint64(v))
	case int16:
		if v < 0 {
			return idInfo{}, fmt.Errorf("negative identifier %d not supported", v)
		}
		return newIDInfoFromUint(uint64(v))
	case int32:
		if v < 0 {
			return idInfo{}, fmt.Errorf("negative identifier %d not supported", v)
		}
		return newIDInfoFromUint(uint64(v))
	case int64:
		if v < 0 {
			return idInfo{}, fmt.Errorf("negative identifier %d not supported", v)
		}
		return newIDInfoFromUint(uint64(v))
	case int:
		if v < 0 {
			return idInfo{}, fmt.Errorf("negative identifier %d not supported", v)
		}
		return newIDInfoFromUint(uint64(v))
	case string:
		return idInfo{raw: v, node: ast.NewASCIINode(v), key: fmt.Sprintf("S:%s", v)}, nil
	default:
		return idInfo{}, fmt.Errorf("unsupported identifier type %T", value)
	}
}

func newIDInfoFromUint(value uint64) (idInfo, error) {
	byteSize := byteSizeForUint(value)
	node := ast.NewUintNode(byteSize, value)
	return idInfo{raw: value, node: node, key: fmt.Sprintf("N:%d", value)}, nil
}

func newIDInfoFromNode(node ast.ItemNode) (idInfo, error) {
	if node == nil {
		return idInfo{}, fmt.Errorf("nil id node")
	}

	switch node.Type() {
	case "ascii":
		str, ok := node.Values().(string)
		if !ok {
			return idInfo{}, fmt.Errorf("unexpected ascii value type %T", node.Values())
		}
		return idInfo{raw: str, node: node, key: fmt.Sprintf("S:%s", str)}, nil
	case "u1", "u2", "u4", "u8":
		values, ok := node.Values().([]uint64)
		if !ok || len(values) == 0 {
			return idInfo{}, fmt.Errorf("invalid unsigned id payload")
		}
		return idInfo{raw: values[0], node: node, key: fmt.Sprintf("N:%d", values[0])}, nil
	case "i1", "i2", "i4", "i8":
		values, ok := node.Values().([]int64)
		if !ok || len(values) == 0 {
			return idInfo{}, fmt.Errorf("invalid signed id payload")
		}
		if values[0] < 0 {
			return idInfo{}, fmt.Errorf("negative identifier %d not supported", values[0])
		}
		uVal := uint64(values[0])
		byteSize := byteSizeForUint(uVal)
		return idInfo{raw: uVal, node: ast.NewUintNode(byteSize, uVal), key: fmt.Sprintf("N:%d", uVal)}, nil
	default:
		return idInfo{}, fmt.Errorf("unsupported id node type %s", node.Type())
	}
}

func byteSizeForUint(value uint64) int {
	switch {
	case value <= 0xFF:
		return 1
	case value <= 0xFFFF:
		return 2
	case value <= 0xFFFFFFFF:
		return 4
	default:
		return 8
	}
}

func keyForRawID(id interface{}) (string, error) {
	switch v := id.(type) {
	case uint64:
		return fmt.Sprintf("N:%d", v), nil
	case string:
		return fmt.Sprintf("S:%s", v), nil
	default:
		return "", fmt.Errorf("unsupported raw id type %T", id)
	}
}

type idRequest struct {
	info idInfo
	ok   bool
}
