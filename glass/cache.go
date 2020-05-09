package main

import (
	"log"
	"net"
	"time"

	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
)

const (
	iasn     = 1
	isourced = 2
	iroute   = 3
	iorigin  = 4
	iaspath  = 5
	iroa     = 6
)

var (
	maxAge = map[int]time.Duration{
		iasn:     time.Hour * 6,
		isourced: time.Minute * 10,
		iroute:   time.Minute * 1,
		iorigin:  time.Minute * 5,
		iaspath:  time.Minute * 5,
		iroa:     time.Hour * 1,
	}
	maxCache = map[int]int{
		iasn:     100,
		isourced: 100,
		iroute:   100,
		iorigin:  100,
		iaspath:  100,
		iroa:     100,
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
}

type asnAge struct {
	name, loc string
	age       time.Time
}

type totalsAge struct {
	v4, v6 uint32
	time   uint64
	age    time.Time
}

type roaAge struct {
	roa *pb.RoaResponse
	age time.Time
}

type aspathAge struct {
	asn []*pb.Asn
	set []*pb.Asn
	age time.Time
}

type routeAge struct {
	subnet pb.IpAddress
	age    time.Time
}

type originAge struct {
	origin uint32
	age    time.Time
}

type sourcedAge struct {
	prefixes []*pb.IpAddress
	v4       uint32
	v6       uint32
	age      time.Time
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
	}
}

// checkTotalCache will check the local cache.
// Only returns the cache entry if it's within 5 minutes
func (s *server) checkTotalCache() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check cache for Totals")

	// If cache entry exists, return true only if the cache entry is still valid.
	if (s.totalCache != totalsAge{}) {
		log.Printf("Returning cache total if timers is still valid")
		return time.Since(s.totalCache.age) < 5*time.Minute
	}

	return false
}

// updateTotalCache will update the local cache.
func (s *server) updateTotalCache(t *bpb.PrefixCountResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Updating cache for Totals")
	s.totalCache = totalsAge{
		v4:   t.GetActive_4(),
		v6:   t.GetActive_6(),
		time: t.GetTime(),
		age:  time.Now(),
	}
}

// checkOriginCache will return an origin uint32 that matches a previous origin check
// if it's still within maxAge.
func (s *server) checkOriginCache(ip string) (uint32, bool) {
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
	if !ok {
		log.Printf("cache miss for origin %s", ip)
	}
	return 0, false
}

// TODO: ideally origin cache should contain the entire subnet, not just IP.
// Will need to re-do how I have this data
func (s *server) updateOriginCache(ip net.IP, origin uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Adding %s to the origin cache", ip.String())

	s.originCache[ip.String()] = originAge{
		origin: origin,
		age:    time.Now(),
	}
}

// checkASPathCache returns two lists of ASNs. The first if the regular as-path
// while the second represents an as-set, if it exists.
// TODO: ideally origin cache should contain the entire subnet, not just IP.
func (s *server) checkASPathCache(ip string) ([]*pb.Asn, []*pb.Asn, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check as-path cache for %s", ip)

	val, ok := s.aspathCache[ip]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("as-path cache entry exists for %s", ip)
		if time.Since(val.age) < maxAge[iaspath] {
			log.Printf("as-path cache hit for %s", ip)
			return val.asn, val.set, ok
		}
		log.Printf("as-path cache entry too old for %s", ip)
	}
	if !ok {
		log.Printf("as-path cache entry does not exist for %s", ip)
	}
	return nil, nil, false
}

func (s *server) updateASPathCache(ip net.IP, asn []*pb.Asn, set []*pb.Asn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("adding %s to the as-path cache", ip.String())

	s.aspathCache[ip.String()] = aspathAge{
		asn: asn,
		set: set,
		age: time.Now(),
	}
}

// checkROACache will return any cached ROA entry.
// TODO: Again, this should be based on subnet...
func (s *server) checkROACache(ipnet *net.IPNet) (*pb.RoaResponse, bool) {
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
	return nil, false
}

func (s *server) updateROACache(ipnet *net.IPNet, roa *pb.RoaResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("adding %s to the as-path cache", ipnet.String())

	s.roaCache[ipnet.String()] = roaAge{
		roa: roa,
		age: time.Now(),
	}
}

// checkRouteCache will return an ipnet that matches a previous route check
// if it's still within maxAge.
func (s *server) checkRouteCache(ip string) (pb.IpAddress, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("Check route cache for %s", ip)

	val, ok := s.routeCache[ip]

	// only return cache entry if it's within the max age
	if ok {
		log.Printf("cache entry exists for %s", ip)
		if time.Since(val.age) < maxAge[iroute] {
			log.Printf("cache hit for route entry for %s", ip)
			return val.subnet, ok
		}
		log.Printf("cache miss for route %s", ip)
	}
	if !ok {
		log.Printf("cache miss for route %s", ip)
	}

	return pb.IpAddress{}, false
}

func (s *server) updateRouteCache(ip net.IP, subnet *pb.IpAddress) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Adding %s to the route cache", ip.String())

	s.routeCache[ip.String()] = routeAge{
		subnet: *subnet,
		age:    time.Now(),
	}
}

// checkASNCache will check the local cache.
// Only returns the cache entry if it's within the maxAge timer.
func (s *server) checkASNCache(asn uint32) (string, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	log.Printf("check ASN cache for AS%d", asn)

	val, ok := s.asNameCache[asn]

	// Only return cache value if it's within the max age
	if ok {
		log.Printf("cache entry exists for AS%d", asn)
		if time.Since(val.age) < maxAge[iasn] {
			log.Printf("cache hit for AS%d", asn)
			return val.name, val.loc, ok
		}
		log.Printf("cache miss for AS%d", asn)
	}
	if !ok {
		log.Printf("cache miss for AS%d", asn)
	}

	return "", "", false
}

func (s *server) updateASNCache(name, loc string, as uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Adding AS%d: %s to the cache", as, name)
	s.asNameCache[as] = asnAge{
		name: name,
		loc:  loc,
		age:  time.Now(),
	}

}

func (s *server) checkSourcedCache(asn uint32) (*sourcedAge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Printf("Check cache for IPs sourced from %d", asn)

	val, ok := s.sourcedCache[asn]

	if ok {
		log.Printf("Cache entry exists for AS%d", asn)
		if time.Since(val.age) < maxAge[isourced] {
			log.Printf("Cache hit for AS%d", asn)
			return &val, ok
		}
		log.Printf("Cache miss for AS%d", asn)
	}

	if !ok {
		log.Printf("Cache miss for AS%d", asn)
	}

	return nil, false
}

func (s *server) updateSourcedCache(prefixes []*pb.IpAddress, v4, v6, asn uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Updating cache for IPs sourced from %d", asn)

	s.sourcedCache[asn] = sourcedAge{
		prefixes: prefixes,
		v4:       v4,
		v6:       v6,
		age:      time.Now(),
	}

}

func (s *server) clearCache() {
	for {
		time.Sleep(5 * time.Minute)
		log.Printf("Clearing old cache entries")
		s.mu.Lock()

		// ASN cache
		for key, val := range s.asNameCache {
			if time.Since(val.age) > maxAge[iasn] {
				delete(s.asNameCache, key)
			}
		}
		if len(s.asNameCache) > maxCache[iasn] {
			log.Printf("AS name cache full, purging...")
			s.asNameCache = make(map[uint32]asnAge)
		}

		// sourced cache
		for key, val := range s.sourcedCache {
			if time.Since(val.age) > maxAge[isourced] {
				delete(s.sourcedCache, key)

			}
		}
		if len(s.sourcedCache) > maxCache[isourced] {
			log.Printf("sourced cache full, purging...")
			s.sourcedCache = make(map[uint32]sourcedAge)
		}

		// route cache
		for key, val := range s.routeCache {
			if time.Since(val.age) > maxAge[iroute] {
				delete(s.routeCache, key)
			}
		}
		if len(s.routeCache) > maxCache[iroute] {
			log.Printf("route cache full, purging...")
			s.routeCache = make(map[string]routeAge)
		}

		// origin cache
		for key, val := range s.originCache {
			if time.Since(val.age) > maxAge[iorigin] {
				delete(s.originCache, key)
			}
		}
		if len(s.originCache) > maxCache[iorigin] {
			log.Printf("origin cache full, purging...")
			s.originCache = make(map[string]originAge)
		}

		// as-path cache
		for key, val := range s.aspathCache {
			if time.Since(val.age) > maxAge[iaspath] {
				delete(s.aspathCache, key)
			}
		}
		if len(s.aspathCache) > maxCache[iaspath] {
			log.Printf("as-path cache full, purging...")
			s.aspathCache = make(map[string]aspathAge)
		}

		// roa cache
		for key, val := range s.roaCache {
			if time.Since(val.age) > maxAge[iroa] {
				delete(s.roaCache, key)
			}
		}
		if len(s.roaCache) > maxCache[iroa] {
			log.Printf("roa cache full, purging...")
			s.roaCache = make(map[string]roaAge)
		}

		s.mu.Unlock()
	}
}
