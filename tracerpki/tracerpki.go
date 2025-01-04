package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/netip"
	"syscall"
	"time"

	"github.com/mellowdrifter/bgp_infrastructure/proto/tracerpki"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/tracerpki"
	bgpstuff "github.com/mellowdrifter/go-bgpstuff.net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	reqPerMinute     = 60
	startPort        = 33434
	endPort          = 33534 // do i really need this if maxHops == 30 ?
	defaultTimeoutMS = 3000
	maxTTL           = 30
)

var (
	payload   = []byte("@BGPSTUFF.NET[\\]^_")
	deadline  = flag.Int("deadline", 2, "bgstuff.net dial deadline in seconds")
	timeoutMS = flag.Int("timeout", 3000, "timeout in milliseconds")
)

type Args struct {
	Location string
	TCP      bool
	ICMP     bool
	V6       bool
	V4       bool
	MaxTTL   uint
	FirstTTL uint
}

// server is used to implement tracerpki.TraceRPKI
type server struct {
	pb.TraceRPKIServer
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTraceRPKIServer(grpcServer, &server{})
	reflection.Register(grpcServer)

	log.Println("gRPC server is listening on :50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *server) GetTraceRPKI(ctx context.Context, req *tracerpki.TraceRPKIRequest) (*tracerpki.TraceRPKIResponse, error) {
	destAddr, err := getDestinationAddresses(req.GetHost())
	if err != nil {
		return nil, err
	}

	localV4, localV6, err := getSocketAddress()
	if err != nil {
		return nil, err
	}
	log.Printf("local v4 is %s, and local v6 is %s\n", localV4, localV6)

	// TODO: Put this into the server object
	glass := bgpstuff.NewBGPClient(false)

	// Create the UDP sender and the ICMP receiver
	sender, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		panic(err)
	}
	receiver, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(receiver)
	defer syscall.Close(sender)

	// set timeouts
	timeout := syscall.NsecToTimeval(1000000 * int64(*timeoutMS))

	log.Printf("tracerpki to %s (%s), %d hops max, using %s as source\n", req.GetHost(), destAddr, maxTTL, localV4)

	for ttl := uint(1); ttl < maxTTL; ttl++ {
		start := time.Now()
		hop := hop{hop: ttl}

		// set the ttl and timeouts on the socket
		syscall.SetsockoptInt(sender, 0x0, syscall.IP_TTL, int(ttl))
		syscall.SetsockoptTimeval(receiver, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &timeout)
		//err = syscall.SetsockoptInt(sender, syscall.SOL_IP, syscall.IP_RECVERR, 1)
		//if err != nil {
		//	fmt.Errorf("error IP_RECVERR: %v", err)
		//}

		syscall.Bind(receiver, &syscall.SockaddrInet4{Port: startPort, Addr: localV4.As4()})
		syscall.Sendto(sender, []byte{}, 0, &syscall.SockaddrInet4{Port: startPort, Addr: destAddr[0].As4()})
		// Increment the source port each time. We don't have to, but traceroute seems to do this
		// startPort++

		// now for some messages?
		// 52 = packetsize
		p := make([]byte, 52)
		// n = packet size
		_, from, err := syscall.Recvfrom(receiver, p, 0)
		// err can be returned if timeout is reached
		if err != nil {
			fmt.Printf("%d\t*\n", ttl)
			continue
		}
		hop.duration = time.Since(start)
		ip := from.(*syscall.SockaddrInet4).Addr
		hop.address = netip.AddrFrom4(ip)
		// not all addresses will have reverse names. Therefore no need to check for an error. If there is an error, just don't bother displaying a hostname.
		host, _ := net.LookupAddr(hop.address.String())
		if len(host) > 0 {
			hop.rDNS = host[0]
		} else {
			hop.rDNS = "*"
		}

		hop.asNumber = getOrigin(glass, hop.address.String())
		hop.asName = getASName(glass, hop.asNumber)
		hop.rpkiStatus = getRPKIStatus(glass, hop.address.String())

		hop.printHop()

		if ip == destAddr[0].As4() {
			log.Printf("at if ip == destAddr[0]")
			return nil, nil
		}

	}
	fmt.Println("maximum hops reached")
	return nil, nil

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
	return rpki
}

func getSocketAddress() (*netip.Addr, *netip.Addr, error) {
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
			if v6.Compare(netip.Addr{}) == 0 && addr.Addr().Is6() {
				v6 = addr.Addr()
			}
		}
	}
	// fmt.Printf("going to return %s and %s\n", v4, v6)
	return &v4, &v6, nil
}

// getgetDestinationAddresses will return a slice of destination addresses.
// It will always follow as ipv4, ipv6, error.
// Any of these values could be nil
func getDestinationAddresses(dest string) ([]netip.Addr, error) {
	// fmt.Printf("address passed = %s", dest)
	// user may pass in an IP directly
	if addr, err := netip.ParseAddr(dest); err == nil {
		// fmt.Println("user passed in an IP address direcly")
		if addr.Is4() {
			return []netip.Addr{addr, {}}, nil
		}
		if addr.Is6() {
			return []netip.Addr{{}, addr}, nil
		}
	}

	// otherwise resolve the name
	alladdrs, err := net.LookupHost(dest)
	if err != nil {
		fmt.Println(err)
		return []netip.Addr{}, err
	}

	return returnRandomIPsFromLookup(alladdrs), nil
}

// returnRandomIPsFromLookup takes a list of ipv4 and ipv6 addresses and returns a slice of one IPv4 and one IPv6.
// This is chosen randomly. Slice will contain an empty struct literal if there is no ipv4 or no ipv6 addresses.
func returnRandomIPsFromLookup(alladdrs []string) []netip.Addr {
	var v4s, v6s []netip.Addr

	for _, addr := range alladdrs {
		t, _ := netip.ParseAddr(addr)
		if t.Is4() {
			v4s = append(v4s, t)
			continue
		}
		if t.Is6() {
			v6s = append(v6s, t)
		}
	}

	addrs := make([]netip.Addr, 2)
	rand.Seed(time.Now().Unix())

	if len(v4s) > 1 {
		addrs[0] = v4s[rand.Intn(len(v4s))]
	}
	if len(v6s) > 1 {
		addrs[1] = v6s[rand.Intn(len(v6s))]
	}

	if len(v4s) == 1 {
		addrs[0] = v4s[0]
	}
	if len(v6s) == 1 {
		addrs[1] = v6s[0]
	}

	return addrs
}

// send UDP probes
func probeUDP(ttl int, destAddr netip.Addr) error {
	// from here to defer, should this be set inside this function or outside?
	sender, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return err
	}
	defer syscall.Close(sender)

	if err := syscall.SetsockoptInt(sender, 0x0, syscall.IP_TTL, ttl); err != nil {
		return err
	}

	if err := syscall.Sendto(sender, payload, 32, &syscall.SockaddrInet4{Port: startPort, Addr: destAddr.As4()}); err != nil {
		return err
	}

	return nil
}

type hop struct {
	hop        uint
	asNumber   int
	address    netip.Addr
	asName     string
	rDNS       string
	duration   time.Duration
	rpkiStatus string
}

func (h *hop) printHop() {
	fmt.Printf("%d  %s\t(%s)\t%d\t%s\tRPKI: %s\n", h.hop, h.rDNS, h.address, h.asNumber, h.asName, h.rpkiStatus)
}

// UNKNOWN looks confusing, so adjust output to not signed
func unknownToNotSigned(status string) string {
	if status == "UNKNOWN" {
		return "NOT SIGNED"
	}
	return status
}
