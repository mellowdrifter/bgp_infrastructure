package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
)

func getServer() server {
	return server{
		mu:    &sync.RWMutex{},
		cache: getNewCache(),
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

	// check empty cache
	cache, ok := srv.checkInvalidsCache("0")
	if ok {
		t.Errorf("expected empty cache, but got %+v", cache)
	}

	// Add items to cache
	invalid1 := pb.InvalidOriginator{Asn: "1", Ip: []string{"1.2.3.0/24", "11.1.0.0/16"}}
	invalid2 := pb.InvalidOriginator{Asn: "2", Ip: []string{"4.5.6.0/24", "12.1.0.0/16"}}
	invalid3 := pb.InvalidOriginator{Asn: "3", Ip: []string{"7.8.9.0/24", "13.1.0.0/16"}}

	invalids := pb.InvalidResponse{
		Asn: []*pb.InvalidOriginator{
			&invalid1, &invalid2, &invalid3,
		},
		CacheTime: uint64(time.Now().Unix()),
	}

	srv.updateInvalidsCache(invalids)

	// Check entire cache
	got, ok := srv.checkInvalidsCache("0")
	if !ok {
		t.Errorf("Updated cache, but nothing returned")
	}

	// Make sure retrived full cache is the same
	if !reflect.DeepEqual(got, invalids) {
		t.Errorf("Received entry not the same. got %+v, expected %+v", got, invalids)
	}

	// Ensure checking cache for a single existing ASN works
	for i, v := range invalids.GetAsn() {
		t.Run(fmt.Sprintf("AS%s", v.GetAsn()), func(t *testing.T) {
			cache, ok := srv.checkInvalidsCache(fmt.Sprint(i + 1))
			if !ok {
				t.Errorf("Cache missing for item #%d", i+1)
			}
			want := pb.InvalidResponse{Asn: []*pb.InvalidOriginator{v}}
			if !reflect.DeepEqual(cache, want) {
				t.Errorf("got: %+v, but wanted: %+v", got, want)
			}
		})
	}

	// Ensure checking cache for a non-existing ASN returns empty
	got, ok = srv.checkInvalidsCache("100")
	if !ok {
		// Cache should exist, but be empty for ASN 100
		t.Errorf("Cache should exist, but got no cache back")
	}
	if !reflect.DeepEqual(got, pb.InvalidResponse{}) {
		t.Errorf("Should be empty, but got: %+v", got)
	}

}

func TestTotalCache(t *testing.T) {
	srv := getServer()

	// check an empty cache
	cache, ok := srv.checkTotalCache()
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache")
	}

	// insert totals into the cache
	totals := pb.TotalResponse{
		Active_4: 1000,
		Active_6: 500,
		Time:     uint64(time.Now().Unix()),
	}

	srv.updateTotalCache(totals)

	// cache should exist
	cache, ok = srv.checkTotalCache()
	if !ok {
		t.Errorf("should be a totals cache entry, but none found")
	}
	if !reflect.DeepEqual(cache, totals) {
		t.Errorf("got %#v from the cache, but expected %#v", cache, totals)
	}

}

func TestOriginCache(t *testing.T) {
	srv := getServer()

	// check an empty cache
	cache, ok := srv.checkOriginCache("192.168.0.0")
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	// Fill cache and check
	t.Parallel()
	var i uint32
	for i = 0; i < 100; i++ {
		t.Run(fmt.Sprintf("AS%d", i), func(t *testing.T) {
			now := uint64(time.Now().Unix())
			resp := pb.OriginResponse{
				OriginAsn: i,
				Exists:    true,
				CacheTime: now,
			}
			ip := fmt.Sprintf("192.168.%d.0", i)
			srv.updateOriginCache(ip, resp)
			cache, ok := srv.checkOriginCache(ip)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}

		})
	}

}

func TestASPathCache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	cache, ok := srv.checkASPathCache("192.168.0.0")
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	// Fill cache and check
	t.Parallel()
	var i uint32
	for i = 0; i < 100; i++ {
		t.Run(fmt.Sprintf("AS%d", i), func(t *testing.T) {
			now := uint64(time.Now().Unix())
			resp := pb.AspathResponse{
				Asn: []*pb.Asn{
					{
						Asplain: 123,
						Asdot:   "123",
					},
					{
						Asplain: 456,
						Asdot:   "456",
					},
				},
				Set: []*pb.Asn{
					{
						Asplain: 321,
						Asdot:   "321",
					},
					{
						Asplain: 654,
						Asdot:   "654",
					},
				},
				Exists:    true,
				CacheTime: now,
			}
			ip := net.ParseIP(fmt.Sprintf("192.168.%d.0", i))
			srv.updateASPathCache(ip, resp)
			cache, ok := srv.checkASPathCache(ip.String())
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}
		})
	}

}

func TestROACache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	_, ipn, err := net.ParseCIDR("192.168.0.0/24")
	if err != nil {
		t.Error(err)
	}
	cache, ok := srv.checkROACache(ipn)
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	t.Parallel()
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("AS%d", i), func(t *testing.T) {
			now := uint64(time.Now().Unix())
			_, ipnet, err := net.ParseCIDR(fmt.Sprintf("192.168.%d.0/24", i))
			resp := pb.RoaResponse{
				IpAddress: &pb.IpAddress{
					Address: ipnet.IP.String(),
					Mask:    24,
				},
				Status:    1,
				Exists:    true,
				CacheTime: now,
			}
			if err != nil {
				t.Error(err)
			}
			srv.updateROACache(ipnet, resp)
			cache, ok := srv.checkROACache(ipnet)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}
		})
	}
}

func TestRouteCache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	cache, ok := srv.checkRouteCache("192.168.0.0")
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	t.Parallel()
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("AS%d", i), func(t *testing.T) {
			now := uint64(time.Now().Unix())
			ip := fmt.Sprintf("192.168.%d.0", i)
			resp := pb.RouteResponse{
				IpAddress: &pb.IpAddress{Address: ip, Mask: 24},
				Exists:    true,
				CacheTime: now,
			}
			srv.updateRouteCache(ip, resp)
			cache, ok := srv.checkRouteCache(ip)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}
		})
	}
}

func TestLocationCache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	cache, ok := srv.checkLocationCache("AMS")
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	t.Parallel()
	for _, airport := range commonPops {
		t.Run(fmt.Sprintf("Airport %s", airport), func(t *testing.T) {
			resp := pb.LocationResponse{
				City:    "ABC",
				Country: "DEF",
				Lat:     "123",
				Long:    "456",
				Image:   "encoded",
			}
			srv.updateLocationCache(airport, resp)
			cache, ok := srv.checkLocationCache(airport)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}
		})
	}
	// TODO: Add this length check to all!
	if len(srv.locCache) != len(commonPops) {
		t.Errorf("expected a cache length of %d, but actual length is %d", len(commonPops), len(srv.locCache))
	}
}

func TestMapCache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	cache, ok := srv.checkMapCache("123,456")
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	t.Parallel()
	for i := 0; i < 100; i++ {
		loc := fmt.Sprintf("123,45%d", i)
		t.Run(loc, func(t *testing.T) {
			maploc := base64.StdEncoding.EncodeToString([]byte(loc))
			srv.updateMapCache(loc, maploc)
			cache, ok := srv.checkMapCache(loc)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if maploc != cache {
				t.Errorf("got %+v, wanted %+v", cache, maploc)
			}
		})
	}
	if len(srv.mapCache) != 100 {
		t.Errorf("expected a mapcache length of %d, but actual length is %d", 100, len(srv.mapCache))
	}

}

func TestASNCache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	cache, ok := srv.checkASNCache(123)
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	t.Parallel()
	var i uint32
	for i = 1; i < 101; i++ {
		t.Run(fmt.Sprintf("ASN %d", i), func(t *testing.T) {
			now := uint64(time.Now().Unix())
			resp := pb.AsnameResponse{
				AsName:    fmt.Sprintf("corportation of %d", i),
				Exists:    true,
				Locale:    "US",
				CacheTime: now,
			}
			srv.updateASNCache(i, resp)
			cache, ok := srv.checkASNCache(i)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}
		})
	}
	if len(srv.asNameCache) != 100 {
		t.Errorf("expected a namecache length of %d, but actual length is %d", 100, len(srv.asNameCache))
	}
}

func TestSourcedCache(t *testing.T) {
	srv := getServer()
	// check an empty cache
	cache, ok := srv.checkSourcedCache(123)
	if ok {
		t.Errorf("expected an empty cache, but got a non empty cache: %#v", cache)
	}

	t.Parallel()
	var i uint32
	for i = 1; i < 101; i++ {
		t.Run(fmt.Sprintf("ASN %d", i), func(t *testing.T) {
			now := uint64(time.Now().Unix())
			resp := pb.SourceResponse{
				IpAddress: []*pb.IpAddress{
					{
						Address: "192.168.0.0/24",
					},
					{
						Address: "2000::/3",
					},
					{
						Address: "3000::/3",
					},
				},
				Exists:    true,
				V4Count:   1,
				V6Count:   2,
				CacheTime: now,
			}
			srv.updateSourcedCache(i, resp)
			cache, ok := srv.checkSourcedCache(i)
			if !ok {
				t.Error("cache entry expected, but none found")
			}
			if !reflect.DeepEqual(cache, resp) {
				t.Errorf("got %+v, wanted %+v", cache, resp)
			}
		})
	}
	if len(srv.sourcedCache) != 100 {
		t.Errorf("expected a namecache length of %d, but actual length is %d", 100, len(srv.sourcedCache))
	}

}

func TestClearCache(t *testing.T) {
	srv := getServer()

	// Much shortened for testing
	tAge := map[int]time.Duration{
		iasn:      time.Millisecond * 500,
		isourced:  time.Minute * 1,
		iroute:    time.Minute * 1,
		iorigin:   time.Minute * 1,
		iaspath:   time.Minute * 1,
		iroa:      time.Minute * 1,
		ilocation: time.Minute * 1,
		imap:      time.Minute * 1,
		itotal:    time.Minute * 1,
		iinvalids: time.Minute * 1,
	}
	tCache := map[int]int{
		iasn:      10,
		isourced:  10,
		iroute:    10,
		iorigin:   10,
		iaspath:   10,
		iroa:      10,
		ilocation: 10,
		imap:      30,
	}

	// Inject into the cache
	srv.updateASNCache(1, pb.AsnameResponse{AsName: "test"})

	// clearCache will run every 100 milliseconds
	sleepTimer := 100 * time.Millisecond
	go srv.clearCache(sleepTimer, tAge, tCache)

	// Cache entry should still be live
	time.Sleep(time.Millisecond * 200)
	_, ok := srv.checkASNCache(1)
	if !ok {
		t.Errorf("expected cache entry to still be there, but none found")
	}

	// After 1 second, cache entry should be gone
	time.Sleep(1 * time.Second)
	_, ok = srv.checkASNCache(1)
	if ok {
		t.Errorf("expected cache entry to be gone, but was still there")
	}

}
