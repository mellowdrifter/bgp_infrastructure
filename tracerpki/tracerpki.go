package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/tracerpki"
	bgpstuff "github.com/mellowdrifter/go-bgpstuff.net"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	protocolICMP = 1 // ICMP protocol number
	timeout      = 2 * time.Second
	maxTTL       = 30
)

var (
	payload   = []byte("@BGPSTUFF.NET[\\]^_")
	deadline  = flag.Int("deadline", 2, "bgstuff.net dial deadline in seconds")
	timeoutMS = flag.Int("timeout", 3000, "timeout in milliseconds")
)

// server is used to implement tracerpki.TraceRPKI
type server struct {
	glass *bgpstuff.Client
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

	grpcServer := grpc.NewServer()
	glass := bgpstuff.NewBGPClient(false)
	pb.RegisterTraceRPKIServer(grpcServer, &server{
		glass: glass,
	})
	reflection.Register(grpcServer)

	log.Println("gRPC server is listening on :50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

/*
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

		hop.asNumber = getOrigin(s.glass, hop.address.String())
		hop.asName = getASName(s.glass, hop.asNumber)
		hop.rpkiStatus = getRPKIStatus(s.glass, hop.address.String())

		hop.printHop()

		if ip == destAddr[0].As4() {
			log.Printf("at if ip == destAddr[0]")
			return nil, nil
		}

	}
	fmt.Println("maximum hops reached")
	return nil, nil
	_ = traceroute(req.GetHost(), 30)

	return nil, nil

}
*/

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
	fmt.Printf("going to return %s and %s\n", v4, v6)
	return &v4, &v6, nil
}

func (s *server) GetTraceRPKI(ctx context.Context, req *pb.TraceRPKIRequest) (*pb.TraceRPKIResponse, error) {

	// TODO: Split this out into a v4 and v6 version as requested
	// Also, maybe a UDP and ICMP version?
	// Where are we going?
	raddr, err := net.ResolveIPAddr("ip4", req.GetHost())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %v", err)
	}

	// Where are we coming from?
	localV4, _, err := getSocketAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get source address: %v", err)
	}

	var hops []hop
	// TODO: Start TTL should be server specific
	for ttl := 2; ttl <= maxTTL; ttl++ {
		start := time.Now()

		conn, err := icmp.ListenPacket("ip4:icmp", localV4.String())
		if err != nil {
			return nil, fmt.Errorf("failed to listen to packet: %v", err)
		}
		defer conn.Close()

		// Set TTL for the outgoing packet
		if err := conn.IPv4PacketConn().SetTTL(ttl); err != nil {
			return nil, fmt.Errorf("failed to set TTL: %v", err)
		}

		// Create ICMP Echo Request
		message := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  ttl,
				Data: payload,
			},
		}

		messageBytes, err := message.Marshal(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ICMP message: %v", err)
		}

		// Send the ICMP Echo Request
		_, err = conn.WriteTo(messageBytes, &net.IPAddr{IP: raddr.IP})
		if err != nil {
			return nil, fmt.Errorf("failed to write packet: %v", err)
		}

		// Set a read deadline
		conn.SetReadDeadline(time.Now().Add(timeout))

		// Buffer for incoming ICMP response
		buffer := make([]byte, 1500)
		n, peer, err := conn.ReadFrom(buffer)
		if err != nil {
			fmt.Printf("%2d  * * * (timeout)\n", ttl)
			continue
		}

		elapsed := time.Since(start)

		// Parse the response
		parsedMessage, err := icmp.ParseMessage(protocolICMP, buffer[:n])
		if err != nil {
			return nil, fmt.Errorf("failed to parse ICMP message: %v", err)
		}

		if parsedMessage.Type == ipv4.ICMPTypeTimeExceeded {
			fmt.Printf("%2d  %v  %v\n", ttl, peer, elapsed)
			hops = append(hops, hop{hop: ttl, address: peer.String(), time: elapsed})
		} else if parsedMessage.Type == ipv4.ICMPTypeEchoReply {
			fmt.Printf("%2d  %v  %v\n", ttl, peer, elapsed)
			hops = append(hops, hop{hop: ttl, address: peer.String(), time: elapsed})
			fmt.Println("Trace complete.")
			break
		}
	}
	fmt.Printf("Hops: %v\n", hops)

	return printRPKIHops(s.glass, hops)
}

func printRPKIHops(glass *bgpstuff.Client, hops []hop) (*pb.TraceRPKIResponse, error) {
	var rpkiHops []*pb.Hop
	var response pb.TraceRPKIResponse
	for _, h := range hops {
		asNumber := getOrigin(glass, h.address)
		asName := getASName(glass, asNumber)
		rpkiStatus := getRPKIStatus(glass, h.address)
		var rDNS string
		host, _ := net.LookupAddr(h.address)
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
