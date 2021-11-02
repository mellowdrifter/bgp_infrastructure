module github.com/mellowdrifter/bgp_infrastructure/clidecode

go 1.17

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

require github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20211102202924-7df8de3dc760

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20211102204422-52aae794c27f // indirect
	golang.org/x/net v0.0.0-20211101193420-4a448f8816b3 // indirect
	golang.org/x/sys v0.0.0-20211102192858-4dd72447c267 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211102202547-e9cf271f7f2c // indirect
	google.golang.org/grpc v1.42.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
