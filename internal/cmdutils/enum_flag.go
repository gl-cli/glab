package cmdutils

import (
	"fmt"
	"maps"
	"slices"
)

type enumValue struct {
	allowed  map[string]struct{}
	valueRef *string
}

func (e *enumValue) Type() string {
	return "string"
}

func (e *enumValue) String() string {
	return *e.valueRef
}

func (e *enumValue) Set(v string) error {
	_, ok := e.allowed[v]
	if !ok {
		return fmt.Errorf("must be one of %v", slices.Collect(maps.Keys(e.allowed)))
	}
	*e.valueRef = v
	return nil
}

func NewEnumValue(allowed []string, d string, v *string) *enumValue {
	if v == nil {
		panic("the given enum flag value cannot be nil")
	}

	m := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		m[a] = struct{}{}
	}
	*v = d
	return &enumValue{
		allowed:  m,
		valueRef: v,
	}
}
