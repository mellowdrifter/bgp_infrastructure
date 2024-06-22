build:
	GOOS=linux go build -o app
	docker build -t gcr.io/bgpstuff/tweeter .
	rm -f app