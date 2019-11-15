package main

import (
	"net"
	"testing"
)

func TestStringToInt(t *testing.T) {
	tests := []struct {
		desc   string
		number string
		want   int
	}{
		{
			desc:   "test 1",
			number: "1",
			want:   1,
		},
		{
			desc:   "test word",
			number: "word",
			want:   0,
		},
	}
	for _, v := range tests {
		got := stringToInt(v.number)
		if got != v.want {
			t.Errorf("Error on %s. Got %d, Want %d\n", v.desc, got, v.want)
		}
	}
}

func TestAsnToInt(t *testing.T) {
	tests := []struct {
		desc    string
		asnText string
		want    int
	}{
		{
			desc:    "test 1",
			asnText: "AS123",
			want:    123,
		},
		{
			desc:    "test 2",
			asnText: "word",
			want:    0,
		},
	}
	for _, v := range tests {
		got := asnToInt(v.asnText)
		if got != v.want {
			t.Errorf("Error on %s. Got %d, Want %d\n", v.desc, got, v.want)
		}
	}

}

func TestIpv4ToByte(t *testing.T) {
	tests := []struct {
		desc string
		ip   string
		want [4]byte
	}{
		{
			desc: "test 1",
			ip:   "10.0.0.0",
			want: [4]byte{10, 0, 0, 0},
		},
		{
			desc: "test 2",
			ip:   "192.168.255.255",
			want: [4]byte{192, 168, 255, 255},
		},
	}
	for _, v := range tests {
		add := net.ParseIP(v.ip)
		got := ipv4ToByte(add.To4())
		if got != v.want {
			t.Errorf("Error on %s. Got %d, Want %d\n", v.desc, got, v.want)
		}
	}
}
