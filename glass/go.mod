module github.com/mellowdrifter/bgp_infrastructure/glass

go 1.17

replace github.com/mellowdrifter/bgp_infrastructure/clidecode => ../clidecode

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/proto/glass => ../proto/glass

require (
	github.com/mellowdrifter/bgp_infrastructure/clidecode v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20210314151012-b74eabfbd431
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20210314151012-b74eabfbd431
	github.com/mellowdrifter/bgp_infrastructure/proto/glass v0.0.0-00010101000000-000000000000
	github.com/smartystreets/goconvey v1.6.4 // indirect
	google.golang.org/grpc v1.41.0
	googlemaps.github.io/maps v1.3.1
	gopkg.in/ini.v1 v1.62.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.1.2 // indirect
	go.opencensus.io v0.22.3 // indirect
	golang.org/x/net v0.0.0-20210924151903-3ad01bbaa167 // indirect
	golang.org/x/sys v0.0.0-20210925032602-92d5a993a665 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/genproto v0.0.0-20210924002016-3dee208752a0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
