package clidecode

import (
	"reflect"
	"testing"
)

func TestDecodeASPaths(t *testing.T) {
	var tests = []struct {
		Name     string
		path     string
		wantPath []uint32
		wantSet  []uint32
	}{
		{
			Name:     "Single AS",
			path:     "3356 12345",
			wantPath: []uint32{3356, 12345},
		},
		{
			Name:     "Dual AS",
			path:     "3356 12345 9876",
			wantPath: []uint32{3356, 12345, 9876},
		},
		{
			Name:     "Single AS-SET",
			path:     "3356 12345 9876 {1212}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212},
		},
		{
			Name:     "Dual AS-SET",
			path:     "3356 12345 9876 {1212 3434}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212, 3434},
		},
	}

	for _, tc := range tests {
		gotPath, gotSet := decodeASPaths(tc.path)
		if !reflect.DeepEqual(gotPath, tc.wantPath) {
			t.Errorf("Got %v, Wanted %v", gotPath, tc.wantPath)
		}
		if !reflect.DeepEqual(gotSet, tc.wantSet) {
			t.Errorf("Got %v, Wanted %v", gotSet, tc.wantSet)
		}
	}
}

func BenchmarkDecodeASPaths(b *testing.B) {
	var tests = []struct {
		Name     string
		path     string
		wantPath []uint32
		wantSet  []uint32
	}{
		{
			Name:     "Single AS",
			path:     "3356 12345",
			wantPath: []uint32{3356, 12345},
		},
		{
			Name:     "Dual AS",
			path:     "3356 12345 9876",
			wantPath: []uint32{3356, 12345, 9876},
		},
		{
			Name:     "Single AS-SET",
			path:     "3356 12345 9876 {1212}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212},
		},
		{
			Name:     "Dual AS-SET",
			path:     "3356 12345 9876 {1212 3434}",
			wantPath: []uint32{3356, 12345, 9876},
			wantSet:  []uint32{1212, 3434},
		},
	}
	for _, tc := range tests {
		for n := 0; n < b.N; n++ {
			decodeASPaths(tc.path)
		}
	}
}
