module github.com/mellowdrifter/bgp_infrastructure/asname

go 1.17

replace github.com/mellowdrifter/bgp_infrastructure/bgpsql => ../bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/clidecode => ../clidecode

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/proto/glass => ../proto/glass

require (
	github.com/golang/protobuf v1.5.2
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20211103021237-a0791f5e00f8
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20211103021237-a0791f5e00f8
	golang.org/x/text v0.3.7
	google.golang.org/grpc v1.42.0
	gopkg.in/ini.v1 v1.63.2
)

require (
	golang.org/x/net v0.0.0-20211104170005-ce137452f963 // indirect
	golang.org/x/sys v0.0.0-20211103235746-7861aae1554b // indirect
	google.golang.org/genproto v0.0.0-20211104193956-4c6863e31247 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
