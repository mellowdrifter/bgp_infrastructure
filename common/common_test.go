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
