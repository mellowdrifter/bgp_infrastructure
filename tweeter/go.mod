module github.com/mellowdrifter/bgp_infrastructure/tweeter

go 1.17

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/proto/grapher => ../proto/grapher

require (
	github.com/ChimeraCoder/anaconda v2.0.0+incompatible
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20211102204422-52aae794c27f
	github.com/mellowdrifter/bgp_infrastructure/proto/grapher v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.42.0
	gopkg.in/ini.v1 v1.63.2
)

require (
	github.com/ChimeraCoder/tokenbucket v0.0.0-20131201223612-c5a927568de7 // indirect
	github.com/azr/backoff v0.0.0-20160115115103-53511d3c7330 // indirect
	github.com/dustin/go-jsonpointer v0.0.0-20160814072949-ba0abeacc3dc // indirect
	github.com/dustin/gojson v0.0.0-20160307161227-2e71ec9dd5ad // indirect
	github.com/garyburd/go-oauth v0.0.0-20180319155456-bca2e7f09a17 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20211101193420-4a448f8816b3 // indirect
	golang.org/x/sys v0.0.0-20211102192858-4dd72447c267 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211102202547-e9cf271f7f2c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
