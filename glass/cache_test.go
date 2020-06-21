package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
)

func getServer() server {
	return server{
		mu: &sync.RWMutex{},
		cache: cache{
			mapCache:     make(map[string]mapAge),
			sourcedCache: make(map[uint32]sourcedAge),
		},
	}
}

/*
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
}*/

func BenchmarkUpdateMapCache(b *testing.B) {
	f, err := os.Open("washington.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	buffer := new(bytes.Buffer)
	png.Encode(buffer, src)
	str := base64.StdEncoding.EncodeToString(buffer.Bytes())
	srv := getServer()

	// Let's encode that image into the cache.
	for i := 0; i < b.N; i++ {
		srv.updateMapCache("-3", "10", pb.MapResponse{
			Image: str,
		})
	}
}

func TestMapCache(t *testing.T) {
	f, err := os.Open("washington.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	buffer := new(bytes.Buffer)
	png.Encode(buffer, src)
	str := base64.StdEncoding.EncodeToString(buffer.Bytes())
	srv := getServer()

	srv.updateMapCache("-3", "10", pb.MapResponse{
		Image: str,
	})

	resp, ok := srv.checkMapCache("-3", "10")
	if !ok {
		t.Errorf("Cache updated, but nothing returned when checking the cache")
	}
	if strings.Compare(resp.GetImage(), str) != 0 {
		t.Errorf("Cache image and live image do not match")

	}
}
