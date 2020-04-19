package main

import (
	"log"
	"time"

	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
)

const (
	asnage     = 1
	sourcedage = 2
)

var (
	maxAge = map[int]time.Duration{
		asnage:     time.Hour * 6,
		sourcedage: time.Minute * 5,
	}
	maxCache = map[int]int{
		asnage:     100,
		sourcedage: 100,
	}
)

type cache struct {
	totalCache   totalsAge
	asNameCache  map[uint32]asnAge
	sourcedCache map[uint32]sourcedAge
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
	}
}

// checkTotalCache will check the local cache.
// Only returns the cache entry if it's within 5 minutes
func (s *server) checkTotalCache() bool {
	defer s.mu.RUnlock()
	s.mu.RLock()
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
	defer s.mu.Unlock()
	s.mu.Lock()
	log.Printf("Updating cache for Totals")
	s.totalCache = totalsAge{
		v4:   t.GetActive_4(),
		v6:   t.GetActive_6(),
		time: t.GetTime(),
		age:  time.Now(),
	}
}

// checkASNCache will check the local cache.
// Only returns the cache entry if it's within the maxAge timer.
func (s *server) checkASNCache(asn uint32) (string, string, bool) {
	defer s.mu.RUnlock()
	s.mu.RLock()
	log.Printf("Check cache for AS%d", asn)

	val, ok := s.asNameCache[asn]

	// Only return cache value if it's within the max age
	if ok {
		log.Printf("Cache entry exists for AS%d", asn)
		if time.Since(val.age) < maxAge[asnage] {
			log.Printf("Cache hit for AS%d", asn)
			return val.name, val.loc, ok
		}
		log.Printf("Cache miss for AS%d", asn)

	}

	if !ok {
		log.Printf("Cache miss for AS%d", asn)
	}

	return "", "", false
}

// updateCache will add a new cache entry.
func (s *server) updateASNCache(name, loc string, as uint32) {
	defer s.mu.Unlock()
	s.mu.Lock()

	log.Printf("Adding AS%d: %s to the cache", as, name)
	s.asNameCache[as] = asnAge{
		name: name,
		loc:  loc,
		age:  time.Now(),
	}

}

func (s *server) checkSourcedCache(asn uint32) (*sourcedAge, bool) {
	defer s.mu.RUnlock()
	s.mu.RLock()

	log.Printf("Check cache for IPs sourced from %d", asn)

	val, ok := s.sourcedCache[asn]

	if ok {
		log.Printf("Cache entry exists for AS%d", asn)
		if time.Since(val.age) < maxAge[sourcedage] {
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
		time.Sleep(3 * time.Minute)
		log.Printf("Clearing old cache entries")
		s.mu.Lock()

		// AS number to name mappings.
		for key, val := range s.asNameCache {
			if time.Since(val.age) > maxAge[asnage] {
				delete(s.asNameCache, key)
			}
		}
		if len(s.asNameCache) > maxCache[asnage] {
			log.Printf("AS name cache full. Purging")
			s.asNameCache = make(map[uint32]asnAge)
		}

		// IPs sourced by source AS number.
		for key, val := range s.sourcedCache {
			if time.Since(val.age) > maxAge[sourcedage] {
				delete(s.sourcedCache, key)

			}
		}
		if len(s.sourcedCache) > maxCache[sourcedage] {
			log.Printf("Sourced cache full. Purging")
			s.sourcedCache = make(map[uint32]sourcedAge)
		}

		s.mu.Unlock()
	}

}
