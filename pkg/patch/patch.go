package patch

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	OpReplace = "replace"
	OpAdd     = "add"
	OpRemove  = "remove"
	OpTest    = "test"
)

func NewJSONPatch(ops ...JSONPatchOperation) *JSONPatch {
	return &JSONPatch{
		ops: ops,
	}
}

type JSONPatch struct {
	ops []JSONPatchOperation
}

func (p *JSONPatch) Append(ops ...JSONPatchOperation) {
	p.ops = append(p.ops, ops...)
}

func (p *JSONPatch) Len() int {
	return len(p.ops)
}

func (p *JSONPatch) Payload() ([]byte, error) {
	if len(p.ops) == 0 {
		return nil, fmt.Errorf("list of operations is empty")
	}
	return json.Marshal(p.ops)
}

type JSONPatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func NewJSONPatchOperation(op, path string, value interface{}) JSONPatchOperation {
	return JSONPatchOperation{
		Op:    op,
		Path:  path,
		Value: value,
	}
}

func WithAddOp(path string, value interface{}) JSONPatchOperation {
	return NewJSONPatchOperation(OpAdd, path, value)
}

func WithRemoveOp(path string) JSONPatchOperation {
	return NewJSONPatchOperation(OpRemove, path, nil)
}

func WithReplaceOp(path string, value interface{}) JSONPatchOperation {
	return NewJSONPatchOperation(OpReplace, path, value)
}

func WithTestOp(path string, value interface{}) JSONPatchOperation {
	return NewJSONPatchOperation(OpTest, path, value)
}

func EscapeJSONPointer(path string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(path)
}
