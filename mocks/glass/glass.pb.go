package glass

import (
	"context"

	gpb "github.com/mellowdrifter/bgp_infrastructure/proto/glass"
	"google.golang.org/grpc"
)

type lookingGlassClient struct {
	cc *grpc.ClientConn
}

func NewLookingGlassClient() gpb.LookingGlassClient {
	cc := &grpc.ClientConn{}
	return &lookingGlassClient{cc}
}

// origin will return the origin AS number
func (l *lookingGlassClient) Origin(ctx context.Context, in *gpb.OriginRequest, opts ...grpc.CallOption) (*gpb.OriginResponse, error) {
	var asn uint32
	var exists bool

	switch in.IpAddress.GetAddress() {
	case "108.31.0.1":
		asn = 701
		exists = true
	case "8.8.8.8":
		asn = 15169
		exists = true
	case "1.1.1.1":
		asn = 13335
		exists = true
	}
	return &gpb.OriginResponse{
		OriginAsn: asn,
		Exists:    exists,
	}, nil
}

// aspath will return the aspath.
func (l *lookingGlassClient) Aspath(ctx context.Context, in *gpb.AspathRequest, opts ...grpc.CallOption) (*gpb.AspathResponse, error) {
	return &gpb.AspathResponse{}, nil
}

// route will return the full ip route output.
func (l *lookingGlassClient) Route(ctx context.Context, in *gpb.RouteRequest, opts ...grpc.CallOption) (*gpb.RouteResponse, error) {
	var route *gpb.IpAddress
	var exists bool

	switch in.IpAddress.GetAddress() {
	case "108.31.0.1":
		route = &gpb.IpAddress{
			Address: "108.31.0.0",
			Mask:    24,
		}
		exists = true
	case "8.8.8.8":
		route = &gpb.IpAddress{
			Address: "8.8.8.0",
			Mask:    24,
		}
		exists = true
	}
	return &gpb.RouteResponse{
		IpAddress: route,
		Exists:    exists,
	}, nil
}

// asname will return the AS name.
func (l *lookingGlassClient) Asname(ctx context.Context, in *gpb.AsnameRequest, opts ...grpc.CallOption) (*gpb.AsnameResponse, error) {
	var asname, locale string
	var exists bool

	switch in.GetAsNumber() {
	case 701:
		asname = "UUNET"
		locale = "US"
		exists = true
	case 15169:
		asname = "GOOGLE"
		locale = "US"
		exists = true
	case 11111:
		asname = "THOMSON-HEALTHCARE"
		locale = "EU"
		exists = true
	}
	return &gpb.AsnameResponse{
		AsName: asname,
		Locale: locale,
		Exists: exists,
	}, nil
}

// roa will return the roa status.
func (l *lookingGlassClient) Roa(ctx context.Context, in *gpb.RoaRequest, opts ...grpc.CallOption) (*gpb.RoaResponse, error) {
	return &gpb.RoaResponse{}, nil
}

// sourced will return all the IPv4 and IPv6 prefixes sources by an AS number
func (l *lookingGlassClient) Sourced(ctx context.Context, in *gpb.SourceRequest, opts ...grpc.CallOption) (*gpb.SourceResponse, error) {
	return &gpb.SourceResponse{}, nil
}

// totals will return the current IPv4 and IPv6 BGP count.
func (l *lookingGlassClient) Totals(ctx context.Context, in *gpb.Empty, opts ...grpc.CallOption) (*gpb.TotalResponse, error) {
	return &gpb.TotalResponse{}, nil
}

// Total number of ASNs
func (l *lookingGlassClient) TotalAsns(ctx context.Context, in *gpb.Empty, opts ...grpc.CallOption) (*gpb.TotalAsnsResponse, error) {
	return &gpb.TotalAsnsResponse{}, nil
}
