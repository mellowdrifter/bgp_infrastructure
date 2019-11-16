package main

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestIpv6ToByte(t *testing.T) {
	tests := []struct {
		desc string
		ip   string
		want [16]byte
	}{
		{
			desc: "test 1",
			ip:   "2001:db8::",
			want: [16]byte{32, 1, 13, 184, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			desc: "test 2",
			ip:   "2001:db8:abc:123:12ab:9911:abdc:ef12",
			want: [16]byte{32, 1, 13, 184, 10, 188, 1, 35, 18, 171, 153, 17, 171, 220, 239, 18},
		},
	}
	for _, v := range tests {
		add := net.ParseIP(v.ip)
		got := ipv6ToByte(add.To16())
		if got != v.want {
			t.Errorf("Error on %s. Got %d, Want %d\n", v.desc, got, v.want)
		}
	}
}

func TestMakeDiff(t *testing.T) {
	tests := []struct {
		desc   string
		new    []roa
		old    []roa
		serial uint32
		want   serialDiff
	}{
		{
			desc:   "empty, no diff",
			new:    []roa{},
			old:    []roa{},
			serial: 0,
			want: serialDiff{
				oldSerial: 0,
				newSerial: 1,
				delRoa:    nil,
				addRoa:    nil,
				diff:      false,
			},
		}, {
			desc: "one ROA, no diff",
			new: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			old: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			serial: 1,
			want: serialDiff{
				oldSerial: 1,
				newSerial: 2,
				delRoa:    nil,
				addRoa:    nil,
				diff:      false,
			},
		}, {
			desc: "Min mask change",
			new: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 23,
					MaxMask: 32,
					ASN:     123,
				},
			},
			old: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			serial: 1,
			want: serialDiff{
				oldSerial: 1,
				newSerial: 2,
				delRoa: []roa{
					roa{
						Prefix:  "192.168.1.1",
						MinMask: 24,
						MaxMask: 32,
						ASN:     123,
					},
				},
				addRoa: []roa{
					roa{
						Prefix:  "192.168.1.1",
						MinMask: 23,
						MaxMask: 32,
						ASN:     123,
					},
				},
				diff: true,
			},
		}, {
			desc: "Max mask change",
			new: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 31,
					ASN:     123,
				},
			},
			old: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			serial: 1,
			want: serialDiff{
				oldSerial: 1,
				newSerial: 2,
				delRoa: []roa{
					roa{
						Prefix:  "192.168.1.1",
						MinMask: 24,
						MaxMask: 32,
						ASN:     123,
					},
				},
				addRoa: []roa{
					roa{
						Prefix:  "192.168.1.1",
						MinMask: 24,
						MaxMask: 31,
						ASN:     123,
					},
				},
				diff: true,
			},
		}, {
			desc: "ASN change",
			new: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			old: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     1234,
				},
			},
			serial: 1,
			want: serialDiff{
				oldSerial: 1,
				newSerial: 2,
				delRoa: []roa{
					roa{
						Prefix:  "192.168.1.1",
						MinMask: 24,
						MaxMask: 32,
						ASN:     1234,
					},
				},
				addRoa: []roa{
					roa{
						Prefix:  "192.168.1.1",
						MinMask: 24,
						MaxMask: 32,
						ASN:     123,
					},
				},
				diff: true,
			},
		}, {
			desc: "Two ROAs to one",
			new: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			old: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
				roa{
					Prefix:  "2001:db8::",
					MinMask: 32,
					MaxMask: 48,
					ASN:     123,
				},
			},
			serial: 1,
			want: serialDiff{
				oldSerial: 1,
				newSerial: 2,
				delRoa: []roa{
					roa{
						Prefix:  "2001:db8::",
						MinMask: 32,
						MaxMask: 48,
						ASN:     123,
					},
				},
				addRoa: nil,
				diff:   true,
			},
		}, {
			desc: "One ROA to two",
			new: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
				roa{
					Prefix:  "2001:db8::",
					MinMask: 32,
					MaxMask: 48,
					ASN:     123,
				},
			},
			old: []roa{
				roa{
					Prefix:  "192.168.1.1",
					MinMask: 24,
					MaxMask: 32,
					ASN:     123,
				},
			},
			serial: 1,
			want: serialDiff{
				oldSerial: 1,
				newSerial: 2,
				delRoa:    nil,
				addRoa: []roa{
					roa{
						Prefix:  "2001:db8::",
						MinMask: 32,
						MaxMask: 48,
						ASN:     123,
					},
				},
				diff: true,
			},
		},
	}
	for _, v := range tests {
		got := makeDiff(v.new, v.old, v.serial)
		if !cmp.Equal(got, v.want, cmp.AllowUnexported(serialDiff{})) {
			t.Errorf("Error on %s. got %#v, Want %#v\n", v.desc, got, v.want)
		}
	}

}
