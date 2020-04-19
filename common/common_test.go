package common

import (
	"net"
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

func TestUint32ToString(t *testing.T) {
	var tests = []struct {
		name string
		in   uint32
		out  string
	}{
		{
			name: "Regular number to uint32",
			out:  "1",
			in:   uint32(1),
		},
		{
			name: "Largest uint32",
			out:  "4294967295",
			in:   uint32(4294967295),
		},
	}

	for _, tt := range tests {
		actual := Uint32ToString(tt.in)
		if actual != tt.out {
			t.Errorf("Error on %s. Expected %s, got %s", tt.name, tt.out, actual)
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
		{
			name:   "Third test",
			first:  []string{"1"},
			second: []string{},
			out:    []string{"1"},
		},
		{
			name:   "Fourth test",
			first:  []string{},
			second: []string{"1"},
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
		{
			name:   "Third test",
			first:  []string{},
			second: []string{"1"},
		},
	}

	for _, tt := range tests {
		actual := Intersection(tt.first, tt.second)
		if !reflect.DeepEqual(actual, tt.out) {
			t.Errorf("Error on %s. Expected %q, got %q", tt.name, tt.out, actual)

		}
	}
}

func BenchmarkIntersection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Intersection([]string{"29435", "15169", "2257"}, []string{"15169", "3357", "1"})
	}
}

func TestIsPublicIP(t *testing.T) {
	var ips = []struct {
		name   string
		ip     string
		public bool
	}{
		{
			name:   "Low public IPv6 address",
			ip:     "2000::",
			public: true,
		},
		{
			name:   "High IPv6 public address",
			ip:     "3fff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
			public: true,
		},
		{
			name:   "Low non-public IPv6 address",
			ip:     "4000::",
			public: false,
		},
		{
			name:   "Link-local test",
			ip:     "ff80:1234::",
			public: false,
		},
		{
			name:   "Public IPv4",
			ip:     "8.8.4.4",
			public: true,
		},
		{
			name:   "Documentation IPv4",
			ip:     "192.0.2.1",
			public: false,
		},
		{
			name:   "Link-local IPv4",
			ip:     "169.254.0.1",
			public: false,
		},
		{
			name:   "Loopback IPv4",
			ip:     "127.0.0.1",
			public: false,
		},
	}

	for _, tt := range ips {
		ip := net.ParseIP(tt.ip)
		actual := IsPublicIP(ip)
		if actual != tt.public {
			t.Errorf("Error on %s, Expected %v, got %v", tt.name, tt.public, actual)
			continue
		}
	}
}

func TestASPlainToASDot(t *testing.T) {
	var asns = []struct {
		test     int
		asn      uint32
		expected string
	}{
		{
			test:     1,
			asn:      uint32(131702),
			expected: "2.630",
		},
		{
			test:     2,
			asn:      uint32(65536),
			expected: "1.0",
		},
		{
			test:     3,
			asn:      uint32(500),
			expected: "500",
		},
		{
			test:     4,
			asn:      uint32(65546),
			expected: "1.10",
		},
		{
			test:     5,
			asn:      uint32(194534),
			expected: "2.63462",
		},
	}

	for _, tt := range asns {
		actual := ASPlainToASDot(tt.asn)
		if actual != tt.expected {
			t.Errorf("Error on test #%d: Expected %s, but got %s", tt.test, tt.expected, actual)
		}
	}

}

func TestASDotToASPlain(t *testing.T) {
	var asns = []struct {
		test     int
		asn      string
		expected uint32
	}{
		{
			test:     1,
			expected: uint32(131702),
			asn:      "2.630",
		},
		{
			test:     2,
			expected: uint32(65536),
			asn:      "1.0",
		},
		{
			test:     3,
			expected: uint32(500),
			asn:      "500",
		},
		{
			test:     4,
			expected: uint32(65546),
			asn:      "1.10",
		},
		{
			test:     5,
			expected: uint32(194534),
			asn:      "2.63462",
		},
	}

	for _, tt := range asns {
		actual := ASDotToASPlain(tt.asn)
		if actual != tt.expected {
			t.Errorf("Error on test #%d: Expected %d, but got %d", tt.test, tt.expected, actual)
		}
	}

}
