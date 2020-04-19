package main

import (
	"net"
	"sync"
	"testing"
	"time"
)

func getServer() server {
	return server{
		mu:           &sync.RWMutex{},
		sourcedCache: make(map[uint32]sourcedAge),
	}

}

func BenchmarkUpdateSourcedCache(b *testing.B) {
	srv := getServer()
	for i := 0; i < b.N; i++ {
		srv.updateSourcedCache(
			[]net.IPNet{
				net.IPNet{
					IP:   net.IP{1, 2, 3, 4},
					Mask: net.IPMask{24},
				},
				net.IPNet{
					IP:   net.IP{5, 6, 7, 8},
					Mask: net.IPMask{23},
				},
				net.IPNet{
					IP:   net.IP{4, 3, 2, 1},
					Mask: net.IPMask{22},
				},
			},
			[]net.IPNet{
				net.IPNet{
					IP:   net.IP{1, 2, 3, 4, 5, 6, 7, 8},
					Mask: net.IPMask{64},
				},
				net.IPNet{
					IP:   net.IP{5, 6, 7, 8, 9, 1, 2, 3},
					Mask: net.IPMask{48},
				},
				net.IPNet{
					IP:   net.IP{4, 3, 2, 1, 1, 2, 3, 4},
					Mask: net.IPMask{36},
				},
			},
			12345,
		)

	}

}

func BenchmarkCheckSourcedCache(b *testing.B) {
	srv := getServer()
	srv.sourcedCache[12345] = sourcedAge{
		v6: []net.IPNet{
			net.IPNet{
				IP:   net.IP{1, 2, 3, 4, 5, 6, 7, 8},
				Mask: net.IPMask{64},
			},
			net.IPNet{
				IP:   net.IP{5, 6, 7, 8, 9, 1, 2, 3},
				Mask: net.IPMask{48},
			},
			net.IPNet{
				IP:   net.IP{4, 3, 2, 1, 1, 2, 3, 4},
				Mask: net.IPMask{36},
			},
		},
		v4: []net.IPNet{
			net.IPNet{
				IP:   net.IP{1, 2, 3, 4},
				Mask: net.IPMask{24},
			},
			net.IPNet{
				IP:   net.IP{5, 6, 7, 8},
				Mask: net.IPMask{23},
			},
			net.IPNet{
				IP:   net.IP{4, 3, 2, 1},
				Mask: net.IPMask{22},
			},
		},
		age: time.Now(),
	}
	for i := 0; i < b.N; i++ {
		srv.checkSourcedCache(1234)
		srv.checkSourcedCache(12345)
	}
}
