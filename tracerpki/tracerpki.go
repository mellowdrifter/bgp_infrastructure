package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"syscall"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/tracerpki"
	bgpstuff "github.com/mellowdrifter/go-bgpstuff.net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	protocolICMP   = 1  // ICMP protocol number
	protocolICMPv6 = 58 // ICMPv6 protocol number
	maxHops        = 30
	timeout        = 3 * time.Second
	packetSize     = 52
)

var (
	payload  = []byte("@BGPSTUFF.NET[\\]^_")
	deadline = flag.Int("deadline", 2, "bgstuff.net dial deadline in seconds")
)

// server is used to implement tracerpki.TraceRPKI
type server struct {
	glass *bgpstuff.Client
	v4    *netip.Addr
	v6    *netip.Addr
}

type hop struct {
	hop     int
	address string
	time    time.Duration
}

type rpkiHop struct {
	hop        hop
	rDNS       string
	asNumber   int
	asName     string
	rpkiStatus string
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	v4, v6, err := getLocalAddress()
	if err != nil {
		log.Fatalf("Failed to get local address: %v", err)
	}

	grpcServer := grpc.NewServer()
	glass := bgpstuff.NewBGPClient(false)
	pb.RegisterTraceRPKIServer(grpcServer, &server{
		glass: glass,
		v4:    v4,
		v6:    v6,
	})
	reflection.Register(grpcServer)

	log.Printf("gRPC server is listening on :50051. Local v4 is %s, local v6 is %s\n", v4, v6)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func getOrigin(glass *bgpstuff.Client, addr string) int {
	origin, _ := glass.GetOrigin(addr)
	return origin
}

func getASName(glass *bgpstuff.Client, as int) string {
	asname, _ := glass.GetASName(as)
	return asname
}

func getRPKIStatus(glass *bgpstuff.Client, addr string) string {
	// TODO: I need the full route, not just a single IP
	rpki, _ := glass.GetROA(addr)
	if rpki == "UNKNOWN" {
		return "NOT SIGNED"
	}
	return rpki
}

// getSocketAddress returns the local v4 and v6 addresses to use for the traceroute
func getLocalAddress() (*netip.Addr, *netip.Addr, error) {
	iAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, nil, err
	}
	var v4, v6 netip.Addr

	for _, a := range iAddrs {
		addr, err := netip.ParsePrefix(a.String())
		if err != nil {
			return nil, nil, errors.New("unable to choose a source IP address")
		}

		if !addr.Addr().IsLoopback() && addr.Addr().IsGlobalUnicast() {
			// if addr.Addr().IsGlobalUnicast() {
			if v4.Compare(netip.Addr{}) == 0 && addr.Addr().Is4() {
				v4 = addr.Addr()
			}
			if v6.Compare(netip.Addr{}) == 0 && addr.Addr().Is6() && !addr.Addr().IsPrivate() {
				v6 = addr.Addr()
			}
		}
	}
	return &v4, &v6, nil
}

func probe(addr *net.IPAddr, ttl int) (time.Duration, string, error) {
	start := time.Now()

	var sock int
	var err error
	var dst syscall.Sockaddr
	var family, proto int

	if addr.IP.To4() != nil {
		family = syscall.AF_INET
		proto = syscall.IPPROTO_UDP
		sock, err = syscall.Socket(family, syscall.SOCK_DGRAM, proto)
		if err != nil {
			return 0, "", err
		}
		defer syscall.Close(sock)

		err = syscall.SetsockoptInt(sock, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
		if err != nil {
			return 0, "", err
		}

		dst4 := &syscall.SockaddrInet4{Port: 33434}
		copy(dst4.Addr[:], addr.IP.To4())
		dst = dst4
	} else {
		family = syscall.AF_INET6
		proto = syscall.IPPROTO_UDP
		sock, err = syscall.Socket(family, syscall.SOCK_DGRAM, proto)
		if err != nil {
			return 0, "", err
		}
		defer syscall.Close(sock)

		err = syscall.SetsockoptInt(sock, syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, ttl)
		if err != nil {
			return 0, "", err
		}

		dst6 := &syscall.SockaddrInet6{Port: 33434}
		copy(dst6.Addr[:], addr.IP.To16())
		dst = dst6
	}

	if err := syscall.Sendto(sock, make([]byte, packetSize), 0, dst); err != nil {
		return 0, "", err
	}

	icmpProto := syscall.IPPROTO_ICMP
	if family == syscall.AF_INET6 {
		icmpProto = syscall.IPPROTO_ICMPV6
	}

	rcvSock, err := syscall.Socket(family, syscall.SOCK_RAW, icmpProto)
	if err != nil {
		return 0, "", err
	}
	defer syscall.Close(rcvSock)

	if err := syscall.SetsockoptTimeval(rcvSock, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{Sec: 1, Usec: 0}); err != nil {
		return 0, "", err
	}

	buf := make([]byte, 512)
	n, from, err := syscall.Recvfrom(rcvSock, buf, 0)
	if err != nil {
		return 0, "", err
	}

	duration := time.Since(start)

	sender := ""
	switch from := from.(type) {
	case *syscall.SockaddrInet4:
		sender = net.IP(from.Addr[:]).String()
	case *syscall.SockaddrInet6:
		sender = net.IP(from.Addr[:]).String()
	}

	if n == 0 || sender == "" {
		return 0, "", errors.New("invalid response")
	}

	return duration, sender, nil
}

func (s *server) GetTraceRPKI(ctx context.Context, req *pb.TraceRPKIRequest) (*pb.TraceRPKIResponse, error) {

	addr, err := net.ResolveIPAddr("ip", req.GetHost())
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve host: %v\n", err)
	}

	var hops []hop

	fmt.Printf("Tracing route to %s [%s]\n", req.GetHost(), addr.String())
	for ttl := 2; ttl <= maxHops; ttl++ {
		// Also, maybe a UDP and ICMP version?
		elapsed, h, err := probe(addr, ttl)
		if err != nil {
			fmt.Printf("%2d  *\n", ttl)
			hops = append(hops, hop{hop: ttl, address: h, time: elapsed})
		} else {
			fmt.Printf("%2d  %s  %v\n", ttl, h, elapsed)
			hops = append(hops, hop{hop: ttl, address: h, time: elapsed})
			if h == addr.String() {
				break
			}
		}
	}
	fmt.Printf("Hops: %v\n", hops)
	return popultateRPKIData(s.glass, hops)
}

func popultateRPKIData(glass *bgpstuff.Client, hops []hop) (*pb.TraceRPKIResponse, error) {
	var rpkiHops []*pb.Hop
	var response pb.TraceRPKIResponse
	for _, h := range hops {
		log.Printf("Looking up details for hop: %v\n", h)
		log.Println("Checking AS Number")
		asNumber := getOrigin(glass, h.address)
		log.Println("Checking AS Name")
		asName := getASName(glass, asNumber)
		log.Println("Checking RPKI Status")
		rpkiStatus := getRPKIStatus(glass, h.address)
		var rDNS string
		log.Println("Checking reverse DNS")
		host, _ := net.LookupAddr(h.address)
		log.Println("Reverse DNS done")
		if len(host) > 0 {
			rDNS = host[0]
		} else {
			rDNS = "*"
		}
		rpkiHops = append(rpkiHops, &pb.Hop{
			Hop:        uint32(h.hop),
			Ip:         h.address,
			Rtt:        uint32(h.time.Nanoseconds()),
			Rdns:       rDNS,
			AsNumber:   uint32(asNumber),
			AsName:     asName,
			RpkiStatus: rpkiStatus,
		})
	}
	response.Hops = rpkiHops
	for _, rh := range rpkiHops {
		fmt.Printf("%2d  %v  %v  %v  %v  %v %v\n", rh.GetHop(), rh.GetIp(), rh.GetRtt(), rh.GetRdns(), rh.GetAsNumber(), rh.GetAsName(), rh.GetRpkiStatus())
	}
	fmt.Println("All done!")
	return &response, nil
}
