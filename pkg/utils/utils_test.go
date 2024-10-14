package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_PrettyTimeAgo(t *testing.T) {
	cases := map[string]string{
		"1s":         "less than a minute ago",
		"30s":        "less than a minute ago",
		"1m08s":      "about 1 minute ago",
		"15m0s":      "about 15 minutes ago",
		"59m10s":     "about 59 minutes ago",
		"1h10m02s":   "about 1 hour ago",
		"15h0m01s":   "about 15 hours ago",
		"30h10m":     "about 1 day ago",
		"50h":        "about 2 days ago",
		"720h05m":    "about 1 month ago",
		"3000h10m":   "about 4 months ago",
		"8760h59m":   "about 1 year ago",
		"17601h59m":  "about 2 years ago",
		"262800h19m": "about 30 years ago",
	}

	for duration, expected := range cases {
		d, e := time.ParseDuration(duration)
		if e != nil {
			t.Errorf("failed to create a duration: %s", e)
		}

		fuzzy := PrettyTimeAgo(d)
		require.Equal(t, fuzzy, expected, "unexpected fuzzy duration value: %s for %s")
	}
}

func Test_Pluralize(t *testing.T) {
	testCases := []struct {
		name   string
		word   string
		amount int
		want   string
	}{
		{
			name:   "singular",
			word:   "label",
			amount: 1,
			want:   "1 label",
		},
		{
			name:   "plural",
			word:   "label",
			amount: 3,
			want:   "3 labels",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			got := Pluralize(tC.amount, tC.word)
			require.Equal(t, got, tC.want, "Pluralize() got = %s, want = %s")
		})
	}
}

func Test_PresentInStringSlice(t *testing.T) {
	testCases := []struct {
		name   string
		hay    []string
		needle string
		want   bool
	}{
		{"simple true", []string{"foo", "bar", "baz"}, "bar", true},
		{"simple false", []string{"foo", "bar", "baz"}, "qux", false},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			got := PresentInStringSlice(tC.hay, tC.needle)
			require.Equal(t, got, tC.want, "PresentInStringSlice() got = %t, want = %t")
		})
	}
}

func Test_PresentInIntSlice(t *testing.T) {
	testCases := []struct {
		name   string
		hay    []int
		needle int
		want   bool
	}{
		{"simple true", []int{1, 2, 3}, 2, true},
		{"simple false", []int{1, 2, 3}, 4, false},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			got := PresentInIntSlice(tC.hay, tC.needle)
			require.Equal(t, got, tC.want, "PresentInIntSlice() got = %t, want = %t")
		})
	}
}

func Test_SanitizePathName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "A regular filename",
			filename: "cli-v1.22.0.json",
			want:     "/cli-v1.22.0.json",
		},
		{
			name:     "A regular filename in a directory",
			filename: "cli/cli-v1.22.0.json",
			want:     "/cli/cli-v1.22.0.json",
		},
		{
			name:     "A filename with directory traversal",
			filename: "cli-v1.../../22.0.zip",
			want:     "/22.0.zip",
		},
		{
			name:     "A particularly nasty filename",
			filename: "..././..././..././etc/password_file",
			want:     "/.../.../.../etc/password_file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePathWanted := SanitizePathName(tt.filename)
			require.Equal(t, filePathWanted, tt.want, "SanitizePathName() got = %s, want = %s")
		})
	}
}

func Test_CommonElementsInStringSlice(t *testing.T) {
	testCases := []struct {
		name   string
		array1 []string
		array2 []string
		want   []string
	}{
		{
			name:   "simple no matching elements",
			array1: []string{"foo", "bar", "baz"},
			array2: []string{"qux", "quux", "quz"},
			want:   []string{},
		},
		{
			name:   "simple matching elements",
			array1: []string{"foo", "quux", "baz"},
			array2: []string{"qux", "quux", "baz"},
			want:   []string{"quux", "baz"},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			got := CommonElementsInStringSlice(tC.array1, tC.array2)
			require.Equal(t, len(got), len(tC.want), "CommonElementsInStringSlice() size of got (%d) and wanted (%d) arrays differ")
			for i := range got {
				require.Equal(t, got[i], tC.want[i], "CommonElementsInStringSlice() got = %s, want = %s")
			}
		})
	}
}

func Test_Map(t *testing.T) {
	type SomeType struct {
		name string
	}

	type Tests[T1, T2 any] struct {
		name  string
		slice []T1
		fn    func(T1) T2
		want  []T2
	}

	tests := []Tests[any, any]{
		{
			"list of strings",
			[]any{"foo", "bar", "baz"},
			func(e any) any { return e },
			[]any{"foo", "bar", "baz"},
		},
		{
			"list of structs",
			[]any{SomeType{"foo"}, SomeType{"bar"}, SomeType{"baz"}},
			func(e any) any { return e.(SomeType).name },
			[]any{"foo", "bar", "baz"},
		},
		{
			"no elements",
			[]any{},
			func(e any) any { return e },
			[]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Map(tt.slice, tt.fn)
			require.Equal(t, got, tt.want, "Test_Map() want %v; but got %v")
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "invalid empty url",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid url without correct scheme",
			input:    "https:/gitlab.com/group",
			expected: false,
		},
		{
			name:     "ok with correct url",
			input:    "https://gitlab.com/group",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualIsValid := IsValidURL(test.input)
			require.Equal(t, test.expected, actualIsValid, "TestIsValidURL() got = %s, want = %s")
		})
	}
}

func TestPtr(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"string", "GitLab"},
		{"int", 503},
		{"float", 50.3},
		{"time", time.Now()},
		{"struct", struct{}{}},
	}

	for _, tt := range tests {
		require.Equal(t, Ptr(tt.val), &tt.val, "TestPtr() got = %s want = %s")
	}
}
