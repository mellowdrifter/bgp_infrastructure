package fakeglass

import (
	"context"
	"sync"

	cli "github.com/mellowdrifter/bgp_infrastructure/clidecode"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
)

type Server struct {
	router cli.Decoder
	mu     *sync.RWMutex
	bsql   *grpc.ClientConn
}

// TotalAsns will return the total number of course ASNs.
func (s *Server) TotalAsns(ctx context.Context, e *pb.Empty) (*pb.TotalAsnsResponse, error) {
	return &pb.TotalAsnsResponse{}, nil
}

// Origin will return the origin ASN for the active route.
func (s *Server) Origin(ctx context.Context, r *pb.OriginRequest) (*pb.OriginResponse, error) {
	return &pb.OriginResponse{}, nil

}

// Totals will return the current IPv4 and IPv6 FIB.
// Grabs from database as it's updated every 5 minutes.
func (s *Server) Totals(ctx context.Context, e *pb.Empty) (*pb.TotalResponse, error) {
	return &pb.TotalResponse{}, nil
}

// Aspath returns a list of ASNs for an IP address.
func (s *Server) Aspath(ctx context.Context, r *pb.AspathRequest) (*pb.AspathResponse, error) {
	return &pb.AspathResponse{}, nil
}

// Route returns the primary active RIB entry for the requested IP.
func (s *Server) Route(ctx context.Context, r *pb.RouteRequest) (*pb.RouteResponse, error) {
	return &pb.RouteResponse{}, nil
}

// Asname will return the registered name of the ASN. As this isn't in bird directly, will need
// to speak to bgpsql to get information from the database.
func (s *Server) Asname(ctx context.Context, r *pb.AsnameRequest, ...grpc.CallOption) (*pb.AsnameResponse, error) {
	return &pb.AsnameResponse{}, nil
}

// Roa will check the ROA status of a prefix.
func (s *Server) Roa(ctx context.Context, r *pb.RoaRequest) (*pb.RoaResponse, error) {
	return &pb.RoaResponse{}, nil
}

func (s *Server) Sourced(ctx context.Context, r *pb.SourceRequest) (*pb.SourceResponse, error) {
	return &pb.SourceResponse{}, nil
}
