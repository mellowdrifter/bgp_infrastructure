package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
)

const (
	iasn      = 1
	isourced  = 2
	iroute    = 3
	iorigin   = 4
	iaspath   = 5
	iroa      = 6
	ilocation = 7
	imap      = 8
	itotal    = 9
	iinvalids = 10
	iasnames  = 11
)

var (
	maxAge = map[int]time.Duration{
		iasn:      time.Hour * 6,
		isourced:  time.Minute * 10,
		iroute:    time.Minute * 1,
		iorigin:   time.Minute * 5,
		iaspath:   time.Minute * 5,
		iroa:      time.Hour * 1,
		ilocation: time.Hour * 24 * 14,
		imap:      time.Hour * 24 * 14,
		itotal:    time.Minute * 10,
		iinvalids: time.Hour * 1,
		iasnames:  time.Hour * 24,
	}
	maxCache = map[int]int{
		iasn:      100,
		isourced:  100,
		iroute:    100,
		iorigin:   100,
		iaspath:   100,
		iroa:      100,
		ilocation: 100,
		imap:      30,
	}
)

type cache struct {
	totalCache   totalsAge
	asNameCache  map[uint32]asnAge
	sourcedCache map[uint32]sourcedAge
	routeCache   map[string]routeAge
	originCache  map[string]originAge
	aspathCache  map[string]aspathAge
	roaCache     map[string]roaAge
	locCache     map[string]locAge
	mapCache     map[string]mapAge
	invCache     invAge
	asnamesCache namesAge
}

type asnAge struct {
	asn pb.AsnameResponse
	age time.Time
}

type totalsAge struct {
	tot pb.TotalResponse
	age time.Time
}

type namesAge struct {
	names map[uint32]*pb.AsnameResponse
	age   time.Time
}

type invAge struct {
	inv pb.InvalidResponse
	age time.Time
}

type roaAge struct {
	roa pb.RoaResponse
	age time.Time
}

type aspathAge struct {
	path pb.AspathResponse
	age  time.Time
}

type routeAge struct {
	rr  pb.RouteResponse
	age time.Time
}

type originAge struct {
	origin pb.OriginResponse
	age    time.Time
}

type sourcedAge struct {
	sr  pb.SourceResponse
	age time.Time
}

type locAge struct {
	loc pb.LocationResponse
	age time.Time
}

type mapAge struct {
	imap string
	age  time.Time
}

func getNewCache() cache {
	return cache{
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
		asnamesCache: namesAge{},
	}
}

// checkTotalCache will check the local cache.
func (s *server) checkTotalCache() (pb.TotalResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check cache for Totals")

	// If cache entry exists, return true only if the cache entry is still valid.
	if !reflect.DeepEqual(s.totalCache, totalsAge{}) {
		log.Printf("Returning cache total if timers is still valid")
		if time.Since(s.totalCache.age) < maxAge[itotal] {
			return s.totalCache.tot, true
		}
	}

	return pb.TotalResponse{}, false
}

// updateTotalCache will update the local cache.
func (s *server) updateTotalCache(t pb.TotalResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Updating cache for Totals")

	s.totalCache = totalsAge{
		tot: t,
		age: time.Now(),
	}
}

// checkOriginCache will return an origin uint32 that matches a previous origin check
// if it's still within age.
func (s *server) checkOriginCache(ip string) (pb.OriginResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check origin cache for %s", ip)

	val, ok := s.originCache[ip]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("cache entry exists for %s", ip)
		if time.Since(val.age) < maxAge[iorigin] {
			log.Printf("cache hit for origin entry for %s", ip)
			return val.origin, ok
		}
		log.Printf("cache miss for origin %s", ip)
	}

	return pb.OriginResponse{}, false
}

// TODO: ideally origin cache should contain the entire subnet, not just IP.
// Will need to re-do how I have this data
func (s *server) updateOriginCache(ip string, res pb.OriginResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Adding %s to the origin cache", ip)

	s.originCache[ip] = originAge{
		origin: res,
		age:    time.Now(),
	}
}

// checkInvalidsCache will check the local cache.
func (s *server) checkInvalidsCache(asn string) (pb.InvalidResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check cache for Invalids using ASN #%s", asn)

	// If cache entry exists, return true only if the cache entry is still valid.
	if time.Since(s.invCache.age) < maxAge[iinvalids] {
		// Empty query means all invalids
		if asn == "0" {
			return s.invCache.inv, true
		}
		// Otherwise only return the specific ASN invalids
		for _, v := range s.invCache.inv.GetAsn() {
			if v.GetAsn() == asn {
				return pb.InvalidResponse{
					Asn: []*pb.InvalidOriginator{
						{
							Asn: v.GetAsn(),
							Ip:  v.GetIp(),
						},
					},
				}, true
			}
		}
		// If cache is fresh, but missing ASN, then we return an empty response, but the cache
		// does exist.
		return pb.InvalidResponse{}, true
	}

	return pb.InvalidResponse{}, false
}

// updateInvalidsCache will update the local cache.
func (s *server) updateInvalidsCache(t pb.InvalidResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Updating cache for Invalids")

	s.invCache = invAge{
		inv: t,
		age: time.Now(),
	}
}

// checkASPathCache returns an AS path response which can contain
// both a list of ASNs plus an AS-SET.
// TODO: ideally origin cache should contain the entire subnet, not just IP.
func (s *server) checkASPathCache(ip string) (pb.AspathResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check as-path cache for %s", ip)

	val, ok := s.aspathCache[ip]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("as-path cache entry exists for %s", ip)
		if time.Since(val.age) < maxAge[iaspath] {
			log.Printf("as-path cache hit for %s", ip)
			return val.path, ok
		}
		log.Printf("as-path cache entry too old for %s", ip)
	}
	if !ok {
		log.Printf("as-path cache entry does not exist for %s", ip)
	}
	return pb.AspathResponse{}, false
}

func (s *server) updateASPathCache(ip net.IP, path pb.AspathResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("adding %s to the as-path cache", ip.String())

	s.aspathCache[ip.String()] = aspathAge{
		path: path,
		age:  time.Now(),
	}
}

// checkROACache will return any cached ROA entry.
// TODO: Again, this should be based on subnet...
func (s *server) checkROACache(ipnet *net.IPNet) (pb.RoaResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check ROA cache for %s", ipnet.String())

	// only return cache if it's within the max age
	val, ok := s.roaCache[ipnet.String()]
	if ok {
		log.Printf("roa cache entry exists for %s", ipnet.String())
		if time.Since(val.age) < maxAge[iroa] {
			log.Printf("roa cache hit for %s", ipnet.String())
			return val.roa, ok
		}
		log.Printf("roa cache entry too old for %s", ipnet.String())
	}
	if !ok {
		log.Printf("roa cache entry does not exist for %s", ipnet.String())
	}
	return pb.RoaResponse{}, false
}

func (s *server) updateROACache(ipnet *net.IPNet, roa pb.RoaResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("adding %v to the as-path cache", ipnet.String())

	s.roaCache[ipnet.String()] = roaAge{
		roa: roa,
		age: time.Now(),
	}
}

// checkRouteCache will return an ipnet that matches a previous route check
// if it's still within age.
func (s *server) checkRouteCache(ip string) (pb.RouteResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check route cache for %s", ip)

	val, ok := s.routeCache[ip]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("cache entry exists for %s", ip)
		if time.Since(val.age) < maxAge[iroute] {
			log.Printf("cache hit for route entry for %s", ip)
			return val.rr, ok
		}
		log.Printf("cache miss for route %s", ip)
	}
	if !ok {
		log.Printf("cache miss for route %s", ip)
	}

	return pb.RouteResponse{}, false
}

func (s *server) updateRouteCache(ip string, rr pb.RouteResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Adding %s to the route cache", ip)

	s.routeCache[ip] = routeAge{
		rr:  rr,
		age: time.Now(),
	}
}

func (s *server) checkLocationCache(airport string) (pb.LocationResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check location cache for %s", airport)

	val, ok := s.locCache[airport]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("cache entry exists for %s", airport)
		if time.Since(val.age) < maxAge[ilocation] {
			log.Printf("cache hit for route entry for %s", airport)
			return val.loc, ok
		}
		log.Printf("cache miss for location %s", airport)
	}
	if !ok {
		log.Printf("cache miss for location %s", airport)
	}

	return pb.LocationResponse{}, false
}

func (s *server) updateLocationCache(airport string, loc pb.LocationResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("adding %s to the location cache", airport)

	// TODO: Check if cache is full!
	s.locCache[airport] = locAge{
		loc: loc,
		age: time.Now(),
	}
}

func (s *server) checkMapCache(coordinates string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check map cache for %s", coordinates)

	val, ok := s.mapCache[fmt.Sprintf("%s", coordinates)]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("cache entry exists for %s", coordinates)
		if time.Since(val.age) < maxAge[imap] {
			log.Printf("cache hit for route entry for %s", coordinates)
			return val.imap, ok
		}
		log.Printf("cache miss for location %s", coordinates)
	}
	if !ok {
		log.Printf("cache miss for location %s", coordinates)
	}

	return "", false
}

func (s *server) updateMapCache(coordinates string, imap string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("adding %s to the map cache", coordinates)

	s.mapCache[coordinates] = mapAge{
		imap: imap,
		age:  time.Now(),
	}
}

// checkASNCache will check the local cache.
// Only returns the cache entry if it's within the age timer.
func (s *server) checkASNCache(asnum uint32) (pb.AsnameResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("check ASN cache for AS%d", asnum)

	if !reflect.DeepEqual(s.asNameCache, namesAge{}) {
		// If cache is old, update straight away.
		if time.Since(s.asnamesCache.age) >= maxAge[iasnames] {
			// Need to unlock the mutex here as Asnames will attempt to gain lock of it.
			s.mu.RUnlock()
			s.Asnames(context.Background(), &pb.Empty{})
			// Then relock it again here.
			s.mu.RLock()
		}

		val, ok := s.asnamesCache.names[asnum]
		if ok {
			return pb.AsnameResponse{
				AsName: val.AsName,
				Locale: val.Locale,
			}, true
		}
	}
	return pb.AsnameResponse{}, false
}

func (s *server) checkASNSCache() (pb.AsnamesResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("check ASN cache for all asnames")

	// If cache entry exists, return true only if the cache entry is still valid.
	if !reflect.DeepEqual(s.asNameCache, namesAge{}) {
		log.Printf("Returning cached asnames if timers is still valid")
		if time.Since(s.asnamesCache.age) < maxAge[iasnames] {
			names := make([]*pb.AsnumberAsnames, 0, len(s.asnamesCache.names))
			for k, v := range s.asnamesCache.names {
				names = append(names, &pb.AsnumberAsnames{
					AsNumber: k,
					Names:    v,
				})
			}
			return pb.AsnamesResponse{
				Asnumnames: names,
			}, true
		}
	}

	return pb.AsnamesResponse{}, false
}

// updateASNSCache will update the local cache.
func (s *server) updateASNSCache(a []*pb.AsnumberAsnames) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Updating cache for AS Names")

	names := make(map[uint32]*pb.AsnameResponse, len(a))

	for _, v := range a {
		names[v.GetAsNumber()] = v.GetNames()
	}

	s.asnamesCache = namesAge{
		names: names,
		age:   time.Now(),
	}
}

func (s *server) checkSourcedCache(asn uint32) (pb.SourceResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("Check cache for IPs sourced from %d", asn)

	val, ok := s.sourcedCache[asn]

	if ok {
		log.Printf("Cache entry exists for AS%d", asn)
		if time.Since(val.age) < maxAge[isourced] {
			log.Printf("Cache hit for AS%d", asn)
			return val.sr, ok
		}
		log.Printf("Cache miss for AS%d", asn)
	}

	if !ok {
		log.Printf("Cache miss for AS%d", asn)
	}

	return pb.SourceResponse{}, false
}

func (s *server) updateSourcedCache(asn uint32, sr pb.SourceResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Updating cache for IPs sourced from %d", asn)

	s.sourcedCache[asn] = sourcedAge{
		sr:  sr,
		age: time.Now(),
	}
}

func (s *server) clearCache(sleep time.Duration, age map[int]time.Duration, count map[int]int) {
	for {
		time.Sleep(sleep)
		log.Println("***")
		log.Printf("Clearing old cache entries")
		s.mu.Lock()

		// ASN cache
		log.Printf("asn cache is currently length %d", len(s.asNameCache))
		for key, val := range s.asNameCache {
			if time.Since(val.age) > age[iasn] {
				delete(s.asNameCache, key)
			}
		}
		if len(s.asNameCache) > count[iasn] {
			log.Printf("AS name cache full, purging...")
			s.asNameCache = make(map[uint32]asnAge)
		}
		log.Printf("asn cache is now length %d", len(s.asNameCache))

		// sourced cache
		log.Printf("sourced cache is currently length %d", len(s.sourcedCache))
		for key, val := range s.sourcedCache {
			if time.Since(val.age) > age[isourced] {
				delete(s.sourcedCache, key)
			}
		}
		if len(s.sourcedCache) > count[isourced] {
			log.Printf("sourced cache full, purging...")
			s.sourcedCache = make(map[uint32]sourcedAge)
		}
		log.Printf("sourced cache is now length %d", len(s.sourcedCache))

		// route cache
		log.Printf("route cache is currently length %d", len(s.routeCache))
		for key, val := range s.routeCache {
			if time.Since(val.age) > age[iroute] {
				delete(s.routeCache, key)
			}
		}
		if len(s.routeCache) > count[iroute] {
			log.Printf("route cache full, purging...")
			s.routeCache = make(map[string]routeAge)
		}
		log.Printf("route cache is now length %d", len(s.routeCache))

		// origin cache
		log.Printf("origin cache is currently length %d", len(s.originCache))
		for key, val := range s.originCache {
			if time.Since(val.age) > age[iorigin] {
				delete(s.originCache, key)
			}
		}
		if len(s.originCache) > count[iorigin] {
			log.Printf("origin cache full, purging...")
			s.originCache = make(map[string]originAge)
		}
		log.Printf("origin cache is now length %d", len(s.originCache))

		// as-path cache
		log.Printf("as-path cache is currently length %d", len(s.aspathCache))
		for key, val := range s.aspathCache {
			if time.Since(val.age) > age[iaspath] {
				delete(s.aspathCache, key)
			}
		}
		if len(s.aspathCache) > count[iaspath] {
			log.Printf("as-path cache full, purging...")
			s.aspathCache = make(map[string]aspathAge)
		}
		log.Printf("as-path cache is now length %d", len(s.aspathCache))

		// roa cache
		log.Printf("roa cache is currently length %d", len(s.roaCache))
		for key, val := range s.roaCache {
			if time.Since(val.age) > age[iroa] {
				delete(s.roaCache, key)
			}
		}
		if len(s.roaCache) > count[iroa] {
			log.Printf("roa cache full, purging...")
			s.roaCache = make(map[string]roaAge)
		}
		log.Printf("roa cache is now length %d", len(s.roaCache))

		// location cache
		log.Printf("location cache is currently length %d", len(s.locCache))
		for key, val := range s.locCache {
			if time.Since(val.age) > age[ilocation] {
				delete(s.locCache, key)
			}
		}
		if len(s.locCache) > count[ilocation] {
			log.Printf("location cache full, puring...")
			s.locCache = make(map[string]locAge)
		}
		log.Printf("location cache is now length %d", len(s.locCache))

		// map cache
		log.Printf("map cache is currently length %d", len(s.mapCache))
		for key, val := range s.mapCache {
			if time.Since(val.age) > age[imap] {
				delete(s.mapCache, key)
			}
		}
		if len(s.mapCache) > count[imap] {
			log.Printf("map cache full, puring...")
			s.mapCache = make(map[string]mapAge)
		}
		log.Printf("map cache is now length %d", len(s.mapCache))

		// invalids cache
		if time.Since(s.invCache.age) > age[iinvalids] {
			s.invCache = invAge{}
		}

		s.mu.Unlock()
		log.Printf("cache cleared")
		log.Println("***")
	}
}
