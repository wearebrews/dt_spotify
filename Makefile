COMMIT = $(shell git rev-parse HEAD)

build:
	go build -o main main.go
run: build
	./main

test_local: build
	python3 tools/local_testing/run.py

docker:
	docker build -t wearebrews/dtspotify:$(COMMIT) -f dockerfiles/dtspotify.dockerfile .
	docker build -t wearebrews/dtspotify_init -f dockerfiles/dtspotify_init.dockerfile .
docker-push: docker
	docker push wearebrews/dtspotify:$(COMMIT)
	docker push wearebrews/dtspotify_init
