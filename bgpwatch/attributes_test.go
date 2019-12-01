package main

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeASPath(t *testing.T) {
	tests := []struct {
		desc  string
		input []byte
		want  []asnSegment
	}{
		{
			desc:  "Test 1, AS_SEQUENCE",
			input: []byte{0x02, 0x02, 0x00, 0x00, 0x90, 0xec, 0x00, 0x00, 0x19, 0x35},
			want: []asnSegment{
				asnSegment{
					Type: 2,
					ASN:  37100,
				},
				asnSegment{
					Type: 2,
					ASN:  6453,
				},
			},
		},
		{
			desc:  "Test 2, AS_SET",
			input: []byte{0x01, 0x02, 0x00, 0x00, 0xcc, 0x8f, 0x00, 0x04, 0x06, 0x2e},
			want: []asnSegment{
				asnSegment{
					Type: 1,
					ASN:  52367,
				},
				asnSegment{
					Type: 1,
					ASN:  263726,
				},
			},
		},
	}

	for _, test := range tests {
		buf := bytes.NewBuffer(test.input)
		got := decodeASPath(buf)

		if !cmp.Equal(got, test.want) {
			t.Errorf("Test (%s): got %+v, want %+v", test.desc, got, test.want)
		}
	}
}
