COMMIT = $(shell git rev-parse HEAD)

build:
	go build -o main main.go
run: build
	./main

test_local: build
	python3 tools/local_testing/run.py

docker:
	docker build -t wearebrews/dtspotify:$(COMMIT) .
docker-push: docker
	docker push wearebrews/dtspotify:$(COMMIT)
