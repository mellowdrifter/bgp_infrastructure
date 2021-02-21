module github.com/mellowdrifter/bgp_infrastructure/bgpsql

go 1.16

replace github.com/mellowdrifter/bgp_infrastructure/clidecode => ../clidecode

replace github.com/mellowdrifter/bgp_infrastructure/common => ../common

replace github.com/mellowdrifter/bgp_infrastructure/proto/glass => ../proto/glass

replace github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql => ../proto/bgpsql

require (
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/protobuf v1.4.3
	github.com/mattn/go-sqlite3 v1.14.6
	github.com/mellowdrifter/bgp_infrastructure/common v0.0.0-00010101000000-000000000000
	github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql v0.0.0-00010101000000-000000000000
	github.com/smartystreets/goconvey v1.6.4 // indirect
	google.golang.org/grpc v1.35.0
	gopkg.in/ini.v1 v1.62.0
)
