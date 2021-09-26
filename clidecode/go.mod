module github.com/mellowdrifter/bgp_infrastructure/clidecode

go 1.17

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

require (
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20210314151012-b74eabfbd431
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20210314151012-b74eabfbd431 // indirect
	golang.org/x/net v0.0.0-20210924151903-3ad01bbaa167 // indirect
	golang.org/x/sys v0.0.0-20210925032602-92d5a993a665 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210924002016-3dee208752a0 // indirect
	google.golang.org/grpc v1.41.0 // indirect
)
