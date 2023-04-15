module github.com/mellowdrifter/bgp_infrastructure/tweeter

go 1.20

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

replace github.com/mellowdrifter/bgp_infrastructure/proto/grapher => ../proto/grapher

require (
	github.com/ChimeraCoder/anaconda v2.0.0+incompatible
	github.com/mattn/go-mastodon v0.0.6
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/proto/grapher v0.0.0-00010101000000-000000000000
	github.com/michimani/gotwi v0.13.0
	google.golang.org/grpc v1.54.0
	gopkg.in/ini.v1 v1.67.0
)

require (
	github.com/ChimeraCoder/tokenbucket v0.0.0-20131201223612-c5a927568de7 // indirect
	github.com/azr/backoff v0.0.0-20160115115103-53511d3c7330 // indirect
	github.com/dustin/go-jsonpointer v0.0.0-20160814072949-ba0abeacc3dc // indirect
	github.com/dustin/gojson v0.0.0-20160307161227-2e71ec9dd5ad // indirect
	github.com/garyburd/go-oauth v0.0.0-20180319155456-bca2e7f09a17 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
