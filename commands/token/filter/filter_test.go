package filter

import (
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	type args[T any] struct {
		s    []T
		test func(t T) bool
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want []T
	}

	type accessToken struct {
		Active bool
		Name   string
	}

	tokens := []accessToken{
		{Active: false, Name: "Token1"},
		{Active: true, Name: "Token1"},
		{Active: false, Name: "Token2"},
		{Active: true, Name: "Token2"},
		{Active: false, Name: "Token3"},
	}

	tests := []testCase[accessToken]{
		{
			name: "find all active tokens",
			args: args[accessToken]{
				s:    tokens,
				test: func(t accessToken) bool { return t.Active },
			},
			want: []accessToken{
				{Active: true, Name: "Token1"},
				{Active: true, Name: "Token2"},
			},
		},
		{
			name: "find active token by name",
			args: args[accessToken]{
				s:    tokens,
				test: func(t accessToken) bool { return t.Active && t.Name == "Token2" },
			},
			want: []accessToken{
				{Active: true, Name: "Token2"},
			},
		},
		{
			name: "find no tokens",
			args: args[accessToken]{
				s:    tokens,
				test: func(t accessToken) bool { return t.Active && t.Name == "Token123" },
			},
			want: []accessToken{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filter(tt.args.s, tt.args.test); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}
