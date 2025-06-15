module github.com/mellowdrifter/clidecode

go 1.18

replace github.com/mellowdrifter/bgp_infrastructure/common => ../bgp_infrastructure/common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../bgp_infrastructure/proto/bgpsql

require github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20220608002136-be36af397d00

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20220608002136-be36af397d00 // indirect
	golang.org/x/net v0.0.0-20220607020251-c690dde0001d // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220607223854-30acc4cbd2aa // indirect
	google.golang.org/grpc v1.47.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)
