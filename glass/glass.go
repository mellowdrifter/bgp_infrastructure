package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mellowdrifter/bgp_infrastructure/common"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"gopkg.in/ini.v1"
)

const (
	maxCache = 100
	maxAge   = time.Hour * 1
)

type server struct {
	cache map[uint32]asnAge
	mu    *sync.RWMutex
}

type asnAge struct {
	name, loc string
	age       time.Time
}

func main() {
	// load in config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}

	logfile := cf.Section("log").Key("logfile").String()

	// Set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	glassServer := &server{
		cache: make(map[uint32]asnAge),
		mu:    &sync.RWMutex{},
	}

	// set up gRPC server
	log.Printf("Listening on port %d\n", 7181)
	lis, err := net.Listen("tcp", ":7181")
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterLookingGlassServer(grpcServer, glassServer)

	grpcServer.Serve(lis)

}

// TODO: Lots of this code is shared with parser. Should create a library which both imports
// Can then also have different libraries for different vendors.
func (s *server) TotalAsns(ctx context.Context, e *pb.Empty) (*pb.TotalAsnsResponse, error) {

	cmd1 := "/usr/sbin/birdc show route primary table master4 | awk '{print $NF}' | tr -d '[]ASie?' | sed -e '1,2d'"
	cmd2 := "/usr/sbin/birdc show route primary table master6 | awk '{print $NF}' | tr -d '[]ASie?' | sed -e '1,2d'"

	as4, err := com.GetOutput(cmd1)
	if err != nil {
		return nil, err
	}
	as6, err := com.GetOutput(cmd2)
	if err != nil {
		return nil, err
	}

	// asNSet is the list of unique source AS numbers in each address family
	as4Set := com.SetListOfStrings(strings.Fields(as4))
	as6Set := com.SetListOfStrings(strings.Fields(as6))

	// as10Set is the total number of unique source AS numbers.
	var as10 []string
	as10 = append(as10, as4Set...)
	as10 = append(as10, as6Set...)
	as10Set := com.SetListOfStrings(as10)

	// as4Only is the set of ASs that only source IPv4
	as4Only := common.InFirstButNotSecond(as4Set, as6Set)

	// as6Only is the set of ASs that only source IPv6
	as6Only := common.InFirstButNotSecond(as6Set, as4Set)

	// asBoth is the set of ASs that source both IPv4 and IPv6
	asBoth := common.Intersection(as4Set, as6Set)

	// TODO: These numbers are weird...
	return &pb.TotalAsnsResponse{
		Total:   uint32(len(asBoth)),
		As4:     uint32(len(as4Set)),
		As6:     uint32(len(as6Set)),
		As10:    uint32(len(as10Set)),
		As4Only: uint32(len(as4Only)),
		As6Only: uint32(len(as6Only)),
	}, nil

}

func (s *server) TotalTransit(ctx context.Context, r *pb.TotalTransitRequest) (*pb.TotalTransitResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
}

// Origin will return the origin ASN for the active route.
func (s *server) Origin(ctx context.Context, r *pb.OriginRequest) (*pb.OriginResponse, error) {
	log.Printf("Running Origin")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asn, exists, err := getOriginFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	if !exists {
		return &pb.OriginResponse{}, nil
	}

	return &pb.OriginResponse{
		OriginAsn: uint32(asn),
		Exists:    exists,
	}, nil

}

func (s *server) Totals(ctx context.Context, e *pb.Empty) (*pb.TotalResponse, error) {
	log.Printf("Running Totals")

	// load in config
	exe, err := os.Executable()
	if err != nil {
		return &pb.TotalResponse{}, errors.New("Unable to load config in Totals")
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		return &pb.TotalResponse{}, fmt.Errorf("failed to read config file: %v", err)
	}

	// gRPC dial the grapher
	bgpinfo := cf.Section("bgpinfo").Key("server").String()
	conn, err := grpc.Dial(bgpinfo, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	b := bpb.NewBgpInfoClient(conn)

	totals, err := b.GetPrefixCount(ctx, &bpb.Empty{})
	if err != nil {
		return &pb.TotalResponse{}, err
	}

	return &pb.TotalResponse{
		Active_4: totals.GetActive_4(),
		Active_6: totals.GetActive_6(),
		Time:     totals.GetTime(),
	}, nil

}

// Aspath returns a list of ASNs for an IP address.
func (s *server) Aspath(ctx context.Context, r *pb.AspathRequest) (*pb.AspathResponse, error) {
	log.Printf("Running Aspath")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	asns, sets, exists, err := getASPathFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	if !exists {
		return &pb.AspathResponse{}, nil
	}

	return &pb.AspathResponse{
		Asn:    asns,
		Set:    sets,
		Exists: exists,
	}, nil
}

// Route returns the primary active RIB entry for the requested IP.
func (s *server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	log.Printf("Running Route")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		return nil, errors.New("Unable to validate IP")
	}

	ipnet, exists, err := getRouteFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	if !exists {
		return &pb.RouteResponse{}, nil
	}

	mask, _ := ipnet.Mask.Size()
	ipaddr := &pb.IpAddress{
		Address: ipnet.IP.String(),
		Mask:    uint32(mask),
	}

	return &pb.RouteResponse{
		IpAddress: ipaddr,
		Exists:    exists,
	}, nil
}

// checkASNCache will check the local cache.
// Only returns the cache entry if it's within the maxAge timer.
func (s *server) checkASNCache(asn uint32) (string, string, bool) {
	defer s.mu.RUnlock()
	s.mu.RLock()
	log.Printf("Check cache for AS%d", asn)

	val, ok := s.cache[asn]

	// Only return cache value if it's within the max age
	if ok {
		log.Printf("Cache entry exists for AS%d", asn)
		if time.Since(val.age) < maxAge {
			log.Printf("Cache entry timer is still valid for AS%d", asn)
			return val.name, val.loc, ok
		}
		log.Printf("Cache entry timer is too old for AS%d", asn)

	}

	if !ok {
		log.Printf("No cache entry found")
	}

	return "", "", false
}

// updateCache will add a new cache entry.
// It'll also purge the cache if we hit the maximum entries.
func (s *server) updateCache(n, l string, as uint32) {
	defer s.mu.Unlock()
	s.mu.Lock()

	// Only store the maxCache entries to prevent a DOS.
	if len(s.cache) > maxCache {
		log.Printf("Max cache entries reached. Purging")
		s.cache = make(map[uint32]asnAge)
	}

	log.Printf("Adding AS%d: %s to the cache", as, n)
	s.cache[as] = asnAge{
		name: n,
		loc:  l,
		age:  time.Now(),
	}

}

// Asname will return the registered name of the ASN. As this isn't in bird directly, will need
// to speak to bgpsql to get information from the database.
func (s *server) Asname(ctx context.Context, r *pb.AsnameRequest) (*pb.AsnameResponse, error) {
	//return nil, grpc.Errorf(codes.Unimplemented, "RPC not yet implemented")
	log.Printf("Running Asname")

	// check local cache first
	n, l, ok := s.checkASNCache(r.GetAsNumber())
	if ok {
		return &pb.AsnameResponse{
			AsName: n,
			Locale: l,
			Exists: true,
		}, nil
	}

	// load in config
	exe, err := os.Executable()
	if err != nil {
		return &pb.AsnameResponse{}, errors.New("Unable to load config in Asname")
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		return &pb.AsnameResponse{}, fmt.Errorf("failed to read config file: %v", err)
	}

	number := bpb.GetAsnameRequest{AsNumber: r.GetAsNumber()}

	// gRPC dial the bgpsql server
	bgpinfo := cf.Section("bgpinfo").Key("server").String()
	conn, err := grpc.Dial(bgpinfo, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	b := bpb.NewBgpInfoClient(conn)

	name, err := b.GetAsname(ctx, &number)
	if err != nil {
		return &pb.AsnameResponse{}, err
	}

	// Cache the result for next time
	s.updateCache(name.GetAsName(), name.GetAsLocale(), r.GetAsNumber())

	return &pb.AsnameResponse{
		AsName: name.GetAsName(),
		Locale: name.GetAsLocale(),
		Exists: name.Exists,
	}, nil

}

// Roa will check the ROA status of a prefix.
// TODO: bird and bird2 do this completely different :(
func (s *server) Roa(ctx context.Context, r *pb.RoaRequest) (*pb.RoaResponse, error) {
	log.Printf("Running Roa")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	status, err := getRoaFromDaemon(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	return status, nil

}

func (s *server) Sourced(ctx context.Context, r *pb.SourceRequest) (*pb.SourceResponse, error) {
	log.Printf("Running Sourced")

	if !com.ValidateASN(r.GetAsNumber()) {
		return nil, errors.New("Invalid AS number")
	}

	subnets, err := getSourcedFromDaemon(r.GetAsNumber())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	if !subnets.Exists {
		return &pb.SourceResponse{}, nil
	}

	return subnets, nil
}

// getSourcedFromDaemon will get all the IPv4 and IPv6 routes sourced from an ASN.
func getSourcedFromDaemon(as uint32) (*pb.SourceResponse, error) {
	log.Printf("Running getSourcedFromDaemon")

	cmd := fmt.Sprintf("/usr/sbin/birdc 'show route primary table master6 where bgp_path ~ [= * %d =]' | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $1}'", as)
	out, err := com.GetOutput(cmd)
	if err != nil {
		return &pb.SourceResponse{}, err
	}
	var prefixes []*pb.IpAddress
	for _, address := range strings.Fields(out) {
		addrMask := strings.Split(address, "/")
		fmt.Printf("%s\n", addrMask)
		prefixes = append(prefixes, &pb.IpAddress{
			Address: addrMask[0],
			Mask:    com.StringToUint32(addrMask[1]),
		})
	}

	v6Count := len(prefixes)

	cmd = fmt.Sprintf("/usr/sbin/birdc 'show route primary table master4 where bgp_path ~ [= * %d =]' | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $1}'", as)
	out, err = com.GetOutput(cmd)
	if err != nil {
		return &pb.SourceResponse{}, err
	}

	for _, address := range strings.Fields(out) {
		addrMask := strings.Split(address, "/")
		prefixes = append(prefixes, &pb.IpAddress{
			Address: addrMask[0],
			Mask:    com.StringToUint32(addrMask[1]),
		})
	}

	v4Count := len(prefixes) - v6Count

	if out == "" {
		return &pb.SourceResponse{}, err

	}
	return &pb.SourceResponse{
		Exists:    true,
		IpAddress: prefixes,
		V4Count:   uint32(v4Count),
		V6Count:   uint32(v6Count),
	}, err

}

// getOriginFromDaemon will get the origin ASN for the passed in IP directly from the BGP daemon.
func getOriginFromDaemon(ip net.IP) (int, bool, error) {
	log.Printf("Running getOriginFromDaemon")

	cmd := fmt.Sprintf("/usr/sbin/birdc show route primary for %s | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $NF}' | tr -d '[]ASie?'", ip.String())
	out, err := com.GetOutput(cmd)
	if err != nil {
		return 0, false, err
	}

	log.Printf(out)

	if strings.Contains("not in table", out) {
		return 0, false, nil
	}

	source, err := strconv.Atoi(out)
	if err != nil {
		return 0, true, err
	}

	return source, true, nil

}

// getASPathFromDaemon will get the ASN list for the passed in IP directly from the BGP daemon.
func getASPathFromDaemon(ip net.IP) ([]*pb.Asn, []*pb.Asn, bool, error) {
	log.Printf("Running getASPathFromDaemon")

	var asns, asSet []*pb.Asn

	cmd := fmt.Sprintf("/usr/sbin/birdc show route primary all for %s | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | grep as_path | awk '{$1=\"\"; print $0}'", ip.String())
	out, err := com.GetOutput(cmd)
	if err != nil {
		return nil, nil, false, err
	}

	log.Printf(out)

	if out == "" {
		return nil, nil, false, nil

	}

	aspath := strings.Fields(out)

	// Need to separate as-set
	var set bool
	for _, as := range aspath {
		if strings.ContainsAny(as, "{}") {
			set = true
			continue
		}

		switch {
		case set == false:
			asns = append(asns, &pb.Asn{
				Asplain: com.StringToUint32(as),
				Asdot:   com.ASPlainToASDot(com.StringToUint32(as)),
			})
		case set == true:
			asns = append(asns, &pb.Asn{
				Asplain: com.StringToUint32(as),
				Asdot:   com.ASPlainToASDot(com.StringToUint32(as)),
			})
		}
	}

	return asns, asSet, true, nil

}

// getRouteFromDaemon will get the prefix for the passed in IP directly from the BGP daemon.
// If network not found, returns false but no error.
func getRouteFromDaemon(ip net.IP) (*net.IPNet, bool, error) {
	log.Printf("Running getRouteFromDaemon")

	cmd := fmt.Sprintf("/usr/sbin/birdc show route primary for %s | grep -Ev 'BIRD|device1|name|info|kernel1|Table' | awk '{print $1}' | tr -d '[]ASie?'", ip.String())
	out, err := com.GetOutput(cmd)
	if err != nil {
		return nil, false, err
	}

	_, net, err := net.ParseCIDR(out)
	if err != nil {
		return nil, false, nil
	}

	return net, true, nil

}

// getRoaFromDaemon will get the ROA status for the requested prefix directly from the BGP daemon.
// TODO: bird vs bird2 is very different :(
func getRoaFromDaemon(ip net.IP) (*pb.RoaResponse, error) {

	// In order to check the ROA, I need the current route.
	prefix, exists, err := getRouteFromDaemon(ip)
	if err != nil || !exists {
		return &pb.RoaResponse{}, err
	}

	// Check for an existing ROA
	// I've set local preference on all routes to make this easier to determine:
	// 200 = ROA_VALID
	// 100 = ROA_UNKNOWN
	//  50 = ROA_INVALID
	statuses := map[string]pb.RoaResponse_ROAStatus{
		"100": pb.RoaResponse_UNKNOWN,
		"50":  pb.RoaResponse_INVALID,
		"200": pb.RoaResponse_VALID,
	}
	cmd := fmt.Sprintf("/usr/sbin/birdc 'show route all primary for %s' | grep local_pref", prefix.String())
	out, err := com.GetOutput(cmd)
	if err != nil {
		return &pb.RoaResponse{}, err
	}

	// Get the local preference
	pref := strings.Fields(out)

	mask, _ := prefix.Mask.Size()
	return &pb.RoaResponse{
		IpAddress: &pb.IpAddress{
			Address: prefix.IP.String(),
			Mask:    uint32(mask),
		},
		Status: statuses[pref[len(pref)-1]],
		Exists: exists,
	}, nil

}
