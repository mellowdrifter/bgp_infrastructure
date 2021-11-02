build:
	gofumpt -w *.go
	go build -o asname *.go
	
cover:
	go test -cover ./...

race:
	go test -race ./...