module github.com/mellowdrifter/bgp_infrastructure/bgpsql

go 1.16

replace github.com/mellowdrifter/bgp_infrastructure/clidecode => ../clidecode

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/glass => ../proto/glass

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/mattn/go-sqlite3 v1.14.6
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-20211102202924-7df8de3dc760
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-20211102204422-52aae794c27f
	google.golang.org/genproto v0.0.0-20211102202547-e9cf271f7f2c // indirect
	google.golang.org/grpc v1.42.0
	gopkg.in/ini.v1 v1.63.2
)
