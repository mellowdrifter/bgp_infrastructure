module github.com/mellowdrifter/bgp_infrastructure/glass

go 1.16

replace github.com/mellowdrifter/bgp_infrastructure/clidecode => ../clidecode

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/proto/glass => ../proto/glass

require (
	github.com/mellowdrifter/bgp_infrastructure/clidecode v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/proto/glass v0.0.0-00010101000000-000000000000
	github.com/smartystreets/goconvey v1.6.4 // indirect
	google.golang.org/grpc v1.35.0
	googlemaps.github.io/maps v1.3.1
	gopkg.in/ini.v1 v1.62.0
)
