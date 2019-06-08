package common

import (
	"reflect"
	"testing"
)

func TestStringToUint32(t *testing.T) {
	var tests = []struct {
		name string
		in   string
		out  uint32
	}{
		{
			name: "Regular number to uint32",
			in:   "1",
			out:  uint32(1),
		},
		{
			name: "Largest uint32",
			in:   "4294967295",
			out:  uint32(4294967295),
		},
		{
			name: "One larger",
			in:   "4294967296",
		},
	}

	for _, tt := range tests {
		actual := StringToUint32(tt.in)
		if actual != tt.out {
			t.Errorf("Error on %s. Expected %d, got %d", tt.name, tt.out, actual)
		}
	}

}

func TestInFirstButNotSecond(t *testing.T) {
	var tests = []struct {
		name   string
		first  []string
		second []string
		out    []string
	}{
		{
			name:   "First test",
			first:  []string{"a", "b", "c"},
			second: []string{"a", "b", "d"},
			out:    []string{"c"},
		},
		{
			name:   "Second test",
			first:  []string{"29435", "15169", "2257"},
			second: []string{"15169", "3357", "1"},
			out:    []string{"29435", "2257"},
		},
	}

	for _, tt := range tests {
		actual := InFirstButNotSecond(tt.first, tt.second)
		if !reflect.DeepEqual(actual, tt.out) {
			t.Errorf("Error on %s. Expected %q, got %q", tt.name, tt.out, actual)

		}
	}
}

func TestIntersection(t *testing.T) {
	var tests = []struct {
		name   string
		first  []string
		second []string
		out    []string
	}{
		{
			name:   "First test",
			first:  []string{"a", "b", "c"},
			second: []string{"a", "b", "d"},
			out:    []string{"a", "b"},
		},
		{
			name:   "Second test",
			first:  []string{"29435", "15169", "2257"},
			second: []string{"15169", "3357", "1"},
			out:    []string{"15169"},
		},
	}

	for _, tt := range tests {
		actual := Intersection(tt.first, tt.second)
		if !reflect.DeepEqual(actual, tt.out) {
			t.Errorf("Error on %s. Expected %q, got %q", tt.name, tt.out, actual)

		}
	}
}
