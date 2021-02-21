module github.com/mellowdrifter/bgp_infrastructure/clidecode

go 1.16

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

require (
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-00010101000000-000000000000 // indirect
	google.golang.org/grpc v1.35.0 // indirect
)
