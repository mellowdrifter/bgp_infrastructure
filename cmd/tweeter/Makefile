build:
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o app .
	docker build -t gcr.io/bgpstuff/tweeter:latest .
	rm -f app

push: build
	docker push gcr.io/bgpstuff/tweeter:latest

deploy: push
	gcloud run deploy tweeter \
		--image gcr.io/bgpstuff/tweeter:latest \
		--platform managed \
		--region us-central1 \
		--project bgpstuff \
		--set-secrets=/config/config.ini=tweeter-config:latest