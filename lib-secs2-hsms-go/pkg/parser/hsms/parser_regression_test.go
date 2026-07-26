package hsms

import (
	"testing"

	"github.com/younglifestyle/secs4go/lib-secs2-hsms-go/pkg/ast"
)

func TestParseMultiByteItemLength(t *testing.T) {
	values := make([]interface{}, 300)
	for i := range values {
		values[i] = byte(i)
	}
	original := ast.NewHSMSDataMessage("", 1, 2, 0, "H<->E", ast.NewBinaryNode(values...), 1, []byte{1, 2, 3, 4})
	parsed, ok := Parse(original.ToBytes())
	if !ok {
		t.Fatal("failed to parse a valid 300-byte item")
	}
	message := parsed.(*ast.DataMessage)
	body, err := message.Get()
	if err != nil {
		t.Fatal(err)
	}
	got := body.(*ast.BinaryNode).Values().([]int)
	if len(got) != len(values) {
		t.Fatalf("parsed %d values, want %d", len(got), len(values))
	}
}

func TestParseRejectsTrailingMessageText(t *testing.T) {
	original := ast.NewHSMSDataMessage("", 1, 2, 0, "H<->E", ast.NewBinaryNode(1), 1, []byte{1, 2, 3, 4}).ToBytes()
	original = append(original, 0)
	original[3]++
	if _, ok := Parse(original); ok {
		t.Fatal("parser accepted trailing bytes after the top-level item")
	}
}
