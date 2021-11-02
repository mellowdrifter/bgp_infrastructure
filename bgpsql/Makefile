build:
	gofumpt -w *.go
	go build -o  bgpsql *.go
	
cover:
	go test -cover ./...

race:
	go test -race ./...
