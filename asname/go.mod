module github.com/mellowdrifter/bgp_infrastructure/asname

go 1.17

replace github.com/mellowdrifter/bgp_infrastructure/bgpsql => ../bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/clidecode => ../clidecode

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/proto/glass => ../proto/glass

require (
	github.com/golang/protobuf v1.5.2
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20211103003410-146782546999
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20211102210844-d203eb202fce
	golang.org/x/text v0.3.7
	google.golang.org/grpc v1.42.0
	gopkg.in/ini.v1 v1.63.2
)

require (
	golang.org/x/net v0.0.0-20211101193420-4a448f8816b3 // indirect
	golang.org/x/sys v0.0.0-20211102192858-4dd72447c267 // indirect
	google.golang.org/genproto v0.0.0-20211102202547-e9cf271f7f2c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
