package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"image/png"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"googlemaps.github.io/maps"

	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	cli "github.com/mellowdrifter/bgp_infrastructure/clidecode"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
	"gopkg.in/ini.v1"
)

type server struct {
	router  cli.Decoder
	mu      *sync.RWMutex
	bsql    *grpc.ClientConn
	bgprpc  string
	airFile string
	mapi    string
	cache
}

var (
	// commonPops are the most used ingress points.
	commonPops = []string{
		"AMS",
		"CDG",
		"FRA",
		"IAD",
		"LHR",
		"ORD",
		"SEA",
	}
)

func main() {
	// load in config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	airFile := fmt.Sprintf("%s/airports/airports.dat", path.Dir(exe))
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}

	logfile := cf.Section("log").Key("logfile").String()
	mapi := cf.Section("local").Key("mapsAPI").String()

	// Set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(f)

	daemon := cf.Section("local").Key("daemon").String()

	var router cli.Decoder
	switch daemon {
	case "bird2":
		router = cli.Bird2Conn{}
	default:
		log.Fatalf("daemon type must be specified")
	}

	bgprpc := cf.Section("bgpsql").Key("server").String()
	conn, err := dialGRPC(bgprpc)
	if err != nil {
		log.Fatalf("Unable to dial gRPC server: %v", err)
	}
	defer conn.Close()

	glassServer := &server{
		router:  router,
		mu:      &sync.RWMutex{},
		bsql:    conn,
		bgprpc:  bgprpc,
		airFile: airFile,
		mapi:    mapi,
		cache:   getNewCache(),
	}

	// set up gRPC server
	log.Printf("Listening on port %d\n", 7181)
	lis, err := net.Listen("tcp", ":7181")
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterLookingGlassServer(grpcServer, glassServer)

	go glassServer.clearCache()

	glassServer.warmCache()

	grpcServer.Serve(lis)

}

func dialGRPC(srv string) (*grpc.ClientConn, error) {
	// Set keepalive on the client
	var kacp = keepalive.ClientParameters{
		Time:    10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout: 3 * time.Second,  // wait 3 seconds for ping ack before considering the connection dead
	}

	log.Printf("Dialling %s\n", srv)
	return grpc.Dial(
		srv,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(kacp),
	)
}

// TotalAsns will return the total number of course ASNs.
func (s *server) TotalAsns(ctx context.Context, e *pb.Empty) (*pb.TotalAsnsResponse, error) {
	log.Printf("Running TotalAsns")

	as, err := s.router.GetTotalSourceASNs()
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.TotalAsnsResponse{}, err
	}

	return &pb.TotalAsnsResponse{
		As4:     as.As4,
		As6:     as.As6,
		As10:    as.As10,
		As4Only: as.As4Only,
		As6Only: as.As6Only,
		AsBoth:  as.AsBoth,
	}, nil

}

// Origin will return the origin ASN for the active route.
func (s *server) Origin(ctx context.Context, r *pb.OriginRequest) (*pb.OriginResponse, error) {
	log.Printf("Running Origin")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.OriginResponse{}, err
	}

	// check local cache
	cache, ok := s.checkOriginCache(r)
	if ok {
		return cache, nil
	}

	origin, exists, err := s.router.GetOriginFromIP(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.OriginResponse{}, err
	}
	if !exists {
		return &pb.OriginResponse{}, err
	}

	resp := &pb.OriginResponse{
		OriginAsn: origin,
		Exists:    exists,
	}

	// update the local cache
	s.updateOriginCache(r, resp)

	return resp, nil
}

// Invalids returns all the ROA invalid prefixes for an ASN. If the ASN passed in = 0,
// then all ASNs advertising invalids is returned.
func (s *server) Invalids(ctx context.Context, r *pb.InvalidsRequest) (*pb.InvalidResponse, error) {
	log.Printf("Running Invalids for ASN %s", r.GetAsn())

	// check local cache
	cache, ok := s.checkInvalidsCache(r.GetAsn())
	if ok {
		return &cache, nil
	}

	inv, err := s.router.GetInvalids()
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.InvalidResponse{}, err
	}

	var resp pb.InvalidResponse
	var invalids []*pb.InvalidOriginator

	for k, v := range inv {
		var src pb.InvalidOriginator
		src.Asn = k
		src.Ip = v

		invalids = append(invalids, &src)
	}
	resp.Asn = invalids

	// update the local cache
	s.updateInvalidsCache(resp)

	// an ASN query of zero means all ASNs.
	if r.GetAsn() == "0" {
		return &resp, nil
	}

	// Otherwise just return the specific ASN and its invalids.
	for _, v := range resp.GetAsn() {
		if v.GetAsn() == r.GetAsn() {
			return &pb.InvalidResponse{
				Asn: []*pb.InvalidOriginator{
					{
						Asn: v.GetAsn(),
						Ip:  v.GetIp(),
					},
				},
			}, nil
		}
	}

	// The ASN queried has no invalids.
	return &pb.InvalidResponse{}, nil

}

// Totals will return the current IPv4 and IPv6 FIB.
// Grabs from database as it's updated every 5 minutes.
func (s *server) Totals(ctx context.Context, e *pb.Empty) (*pb.TotalResponse, error) {
	log.Printf("Running Totals")

	// check local cache first
	if s.checkTotalCache() {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return &s.totalCache.tot, nil
	}

	stub := bpb.NewBgpInfoClient(s.bsql)
	totals, err := stub.GetPrefixCount(ctx, &bpb.Empty{})
	if err != nil {
		s.handleUnavailableRPC(err)
		return &pb.TotalResponse{}, err
	}

	tot := &pb.TotalResponse{
		Active_4: totals.GetActive_4(),
		Active_6: totals.GetActive_6(),
		Time:     totals.GetTime(),
	}

	// update local cache
	s.updateTotalCache(tot)

	return tot, nil
}

// Aspath returns a list of ASNs for an IP address.
func (s *server) Aspath(ctx context.Context, r *pb.AspathRequest) (*pb.AspathResponse, error) {
	log.Printf("Running Aspath")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.AspathResponse{}, err
	}

	// check local cache
	a, st, ok := s.checkASPathCache(ip.String())
	if ok {
		return &pb.AspathResponse{
			Asn:    a,
			Set:    st,
			Exists: true,
		}, nil
	}

	paths, exists, err := s.router.GetASPathFromIP(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.AspathResponse{}, err
	}

	// IP route may not exist. Return no error, but not existing either.
	if !exists {
		return &pb.AspathResponse{}, nil
	}

	// Repackage into proto
	var p = make([]*pb.Asn, 0, len(paths.Path))
	for _, v := range paths.Path {
		p = append(p, &pb.Asn{
			Asplain: v,
			Asdot:   com.ASPlainToASDot(v),
		})
	}

	var set = make([]*pb.Asn, 0, len(paths.Set))
	for _, v := range paths.Set {
		set = append(set, &pb.Asn{
			Asplain: v,
			Asdot:   com.ASPlainToASDot(v),
		})
	}

	// update the cache
	s.updateASPathCache(ip, p, set)

	return &pb.AspathResponse{
		Asn:    p,
		Set:    set,
		Exists: exists,
	}, nil
}

// Route returns the primary active RIB entry for the requested IP.
func (s *server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	log.Printf("Running Route")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		return &pb.RouteResponse{}, errors.New("Unable to validate IP")
	}

	// check local cache first
	ipnetcache, ok := s.checkRouteCache(ip.String())
	if ok {
		return &pb.RouteResponse{
			IpAddress: &ipnetcache,
			Exists:    true,
		}, nil
	}

	ipnet, exists, err := s.router.GetRoute(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.RouteResponse{}, err
	}
	if !exists {
		return &pb.RouteResponse{}, nil
	}

	mask, _ := ipnet.Mask.Size()
	ipaddr := &pb.IpAddress{
		Address: ipnet.IP.String(),
		Mask:    uint32(mask),
	}

	// cache the result
	s.updateRouteCache(ip, ipaddr)

	return &pb.RouteResponse{
		IpAddress: ipaddr,
		Exists:    exists,
	}, nil
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

	number := bpb.GetAsnameRequest{AsNumber: r.GetAsNumber()}

	stub := bpb.NewBgpInfoClient(s.bsql)
	name, err := stub.GetAsname(ctx, &number)
	if err != nil {
		s.handleUnavailableRPC(err)
		return &pb.AsnameResponse{}, err
	}

	// Cache the result for next time
	s.updateASNCache(name.GetAsName(), name.GetAsLocale(), r.GetAsNumber())

	return &pb.AsnameResponse{
		AsName: name.GetAsName(),
		Locale: name.GetAsLocale(),
		Exists: name.Exists,
	}, nil

}

// Asnames will return all of the registered AS number to AS name mappings.
func (s *server) Asnames(ctx context.Context, e *pb.Empty) (*pb.AsnamesResponse, error) {
	log.Printf("Running Asnames")

	stub := bpb.NewBgpInfoClient(s.bsql)
	names, err := stub.GetAsnames(ctx, &bpb.Empty{})
	if err != nil {
		s.handleUnavailableRPC(err)
		return &pb.AsnamesResponse{}, err
	}

	var allnames pb.AsnamesResponse
	// TODO: This is awful. Some of these proto messages should be defined elsewhere to be shared
	for _, v := range names.Asnumnames {
		var t pb.AsnumberAsnames
		// DOES THIS WORK???
		t = pb.AsnumberAsnames(*v)
		//t.AsName = v.AsName
		//t.AsNumber = v.AsNumber
		//t.AsLocale = v.AsLocale
		allnames.Asnumnames = append(allnames.Asnumnames, &t)
	}

	return &allnames, nil
}

// Roa will check the ROA status of a prefix.
func (s *server) Roa(ctx context.Context, r *pb.RoaRequest) (*pb.RoaResponse, error) {
	log.Printf("Running Roa")

	ip, err := com.ValidateIP(r.GetIpAddress().GetAddress())
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.RoaResponse{}, err
	}

	// In oder to check ROA, I first need the FIB entry as well as the current source ASN.
	ipnet, exists, err := s.router.GetRoute(ip)
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.RoaResponse{}, err
	}
	origin, err := s.Origin(ctx, &pb.OriginRequest{IpAddress: r.IpAddress})
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.RoaResponse{}, err
	}

	// TODO: Not sure if I should check cache before?
	// or getroute should be cached itself
	if !exists {
		return &pb.RoaResponse{}, fmt.Errorf("No route exists for %s, so unable to check ROA status", ip.String())
	}

	// check local cache
	roa, ok := s.checkROACache(ipnet)
	if ok {
		return roa, nil
	}

	status, exists, err := s.router.GetROA(ipnet, origin.GetOriginAsn())
	if err != nil {
		log.Printf("Error: %v", err)
		return &pb.RoaResponse{}, err
	}

	// Check for an existing ROA
	// I've set local preference on all routes to make this easier to determine:
	// 200 = ROA_VALID
	// 100 = ROA_UNKNOWN
	//  50 = ROA_INVALID
	statuses := map[int]pb.RoaResponse_ROAStatus{
		cli.RUnknown: pb.RoaResponse_UNKNOWN,
		cli.RInvalid: pb.RoaResponse_INVALID,
		cli.RValid:   pb.RoaResponse_VALID,
	}

	mask, _ := ipnet.Mask.Size()
	resp := &pb.RoaResponse{
		IpAddress: &pb.IpAddress{
			Address: ipnet.IP.String(),
			Mask:    uint32(mask),
		},
		Status: statuses[status],
		Exists: exists,
	}
	// update cache
	s.updateROACache(ipnet, resp)

	return resp, nil
}

func (s *server) Sourced(ctx context.Context, r *pb.SourceRequest) (*pb.SourceResponse, error) {
	log.Printf("Running Sourced")
	defer com.TimeFunction(time.Now(), "Sourced")

	if !com.ValidateASN(r.GetAsNumber()) {
		return &pb.SourceResponse{}, fmt.Errorf("Invalid AS number")
	}

	// check local cache first
	p, ok := s.checkSourcedCache(r.GetAsNumber())
	if ok {
		return &pb.SourceResponse{
			IpAddress: p.prefixes,
			Exists:    true,
			V4Count:   p.v4,
			V6Count:   p.v6,
		}, nil
	}

	v4, err := s.router.GetIPv4FromSource(r.GetAsNumber())
	if err != nil {
		return &pb.SourceResponse{}, fmt.Errorf("Error on getting IPv4 from source: %w", err)
	}

	var prefixes = make([]*pb.IpAddress, 0, len(v4))
	for _, v := range v4 {
		mask, _ := v.Mask.Size()
		prefixes = append(prefixes, &pb.IpAddress{
			Address: v.IP.String(),
			Mask:    uint32(mask),
		})
	}

	v6, err := s.router.GetIPv6FromSource(r.GetAsNumber())
	if err != nil {
		return &pb.SourceResponse{}, fmt.Errorf("Error on getting IPv6 from source: %w", err)
	}

	for _, v := range v6 {
		mask, _ := v.Mask.Size()
		prefixes = append(prefixes, &pb.IpAddress{
			Address: v.IP.String(),
			Mask:    uint32(mask),
		})
	}

	// Update the local cache
	s.updateSourcedCache(prefixes, uint32(len(v4)), uint32(len(v6)), r.GetAsNumber())

	// No prefixes will return empty, but no error
	if len(prefixes) == 0 {
		return &pb.SourceResponse{}, nil
	}

	return &pb.SourceResponse{
		IpAddress: prefixes,
		Exists:    true,
		V4Count:   uint32(len(v4)),
		V6Count:   uint32(len(v6)),
	}, nil
}

// bgpsql server might go offline, if so we should attempt to reconnect.
func (s *server) handleUnavailableRPC(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := status.FromError(err)
	if !ok {
		log.Println("RPC error, but not a status code")
		log.Printf("Error is: %+v\n", err)
	}
	if st.Code() == codes.Unavailable {
		log.Printf("Server not available")
		conn, err := dialGRPC(s.bgprpc)
		if err != nil {
			log.Printf("Still unable to reconnect to gRPC server: %v", err)
		}
		s.bsql = conn
	}
}

// Location will attempt to return the city, country, and lat/long co-ordinates from an airport code.
func (s *server) Location(ctx context.Context, r *pb.LocationRequest) (*pb.LocationResponse, error) {
	log.Printf("Running Location")
	defer com.TimeFunction(time.Now(), "Location")

	// check local cache first
	cloc, ok := s.checkLocationCache(r.GetAirport())
	if ok {
		return &cloc, nil
	}

	f, err := os.Open(s.airFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to open airports data file: %v", err)
	}
	defer f.Close()

	// Get location co-ordinates
	loc, err := getLocation(r.GetAirport(), f)
	if err != nil {
		return nil, fmt.Errorf("Unable to determine location: %v", err)
	}

	// Now get the map
	l, err := s.getMap(ctx, loc)

	// update cache
	s.updateLocationCache(r.GetAirport(), *l)

	return l, nil
}

func getLocation(loc string, file *os.File) (*pb.LocationResponse, error) {
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to parse csv file: %v", err)
	}

	for _, row := range records {
		if row[4] == loc {
			return &pb.LocationResponse{
				City:    row[2],
				Country: row[3],
				Lat:     row[6],
				Long:    row[7],
			}, nil
		}

	}
	return nil, fmt.Errorf("unable to find airport code %s in the records", loc)
}

// warmCache will fill the cache with the most common ingress points.
func (s *server) warmCache() {
	log.Printf("Warming up the cache")

	for _, loc := range commonPops {
		s.Location(context.Background(), &pb.LocationRequest{
			Airport: loc,
		})
	}
	log.Println("Cache filled")
}

// Map adds an image from Google Maps of the co-ordinates and then updates
// the location response with a base64 encoded version of the image.
func (s *server) getMap(ctx context.Context, r *pb.LocationResponse) (*pb.LocationResponse, error) {
	// check local cache first
	cor := fmt.Sprintf("%s%s", r.GetLat(), r.GetLong())
	cmap, ok := s.checkMapCache(cor)
	if ok {
		r.Image = cmap
		return r, nil
	}
	// get the map and encode
	c, err := maps.NewClient(maps.WithAPIKey(s.mapi))
	if err != nil {
		return r, err
	}
	req := maps.StaticMapRequest{
		Center: fmt.Sprintf("%s,%s", r.GetLat(), r.GetLong()),
		Zoom:   9,
		Size:   "500x500",
		Format: maps.Format("png"),
	}
	img, err := c.StaticMap(ctx, &req)
	if err != nil {
		return r, err
	}
	buffer := new(bytes.Buffer)
	png.Encode(buffer, img)

	rmap := base64.StdEncoding.EncodeToString(buffer.Bytes())

	// Update the cache
	s.updateMapCache(cor, rmap)

	r.Image = rmap

	return r, nil
}
