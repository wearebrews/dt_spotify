COMMIT = $(shell git rev-parse HEAD)

build:
	go build -o main main.go
run: build
	./main

test_local: build
	python3 tools/local_testing/run.py

docker:
	docker build -t wearebrews/dtspotify:$(COMMIT) -f dockerfiles/dtspotify.dockerfile .
	docker build -t wearebrews/dtspotify -f dockerfiles/dtspotify.dockerfile .
docker-push: docker
	docker push wearebrews/dtspotify:$(COMMIT)
	docker push wearebrews/dtspotify

docker-push-dev:
	docker build -t wearebrews/dtspotify:dev -f dockerfiles/dtspotify.dockerfile .
	docker push wearebrews/dtspotify:dev