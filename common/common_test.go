package common

import "testing"

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
