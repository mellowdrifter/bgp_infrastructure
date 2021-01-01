package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
)

func getServer() server {
	return server{
		mu: &sync.RWMutex{},
		cache: cache{
			totalCache:   totalsAge{},
			asNameCache:  make(map[uint32]asnAge),
			sourcedCache: make(map[uint32]sourcedAge),
			routeCache:   make(map[string]routeAge),
			originCache:  make(map[string]originAge),
			aspathCache:  make(map[string]aspathAge),
			roaCache:     make(map[string]roaAge),
			locCache:     make(map[string]locAge),
			mapCache:     make(map[string]mapAge),
			invCache:     invAge{},
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
}

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
}*/

func BenchmarkUpdateInvalidsCache(b *testing.B) {
	//t.Parallel()
	srv := getServer()

	invalids := pb.InvalidResponse{
		Asn: []*pb.InvalidOriginator{
			{Asn: "3356", Ip: []string{"1.2.3.0/24", "12.1.0.0/16"}},
			{Asn: "1", Ip: []string{"4.5.6.0/24", "13.1.0.0/16"}},
			{Asn: "2", Ip: []string{"5.6.7.0/24", "14.1.0.0/16"}}},
	}

	for i := 0; i < b.N; i++ {
		srv.updateInvalidsCache(invalids)
	}

}

func TestInvalidsCache(t *testing.T) {
	srv := getServer()

	invalid1 := pb.InvalidOriginator{Asn: "1", Ip: []string{"1.2.3.0/24", "11.1.0.0/16"}}
	invalid2 := pb.InvalidOriginator{Asn: "2", Ip: []string{"4.5.6.0/24", "12.1.0.0/16"}}
	invalid3 := pb.InvalidOriginator{Asn: "3", Ip: []string{"7.8.9.0/24", "13.1.0.0/16"}}

	invalids := pb.InvalidResponse{
		Asn: []*pb.InvalidOriginator{
			&invalid1, &invalid2, &invalid3,
		}}

	srv.updateInvalidsCache(invalids)

	// Check entire cache
	got, ok := srv.checkInvalidsCache("0")
	if !ok {
		t.Errorf("Updated cache, but nothing returned")
	}

	// Make sure retrived full cache is the same
	if !reflect.DeepEqual(got, invalids) {
		t.Errorf("Received entry not the same")
	}

	// Ensure checking cache for a single existing ASN works
	for i, v := range invalids.GetAsn() {
		got, ok := srv.checkInvalidsCache(fmt.Sprint(i + 1))
		if !ok {
			t.Errorf("Cache missing for item #%d", i)
		}
		want := pb.InvalidResponse{Asn: []*pb.InvalidOriginator{v}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %+v, but wanted: %+v", got, want)
		}
	}

	// Ensure checking cache for a non-existing ASN returns empty
	got, ok = srv.checkInvalidsCache("100")
	if ok {
		t.Errorf("Cache should be empty, but it's not")
	}
	if !reflect.DeepEqual(got, pb.InvalidResponse{}) {
		t.Errorf("Should be empty, but got: %+v", got)
	}

}
