.PHONY: test vet build generate

test:
	go test -race -buildvcs=false ./...

vet:
	go vet -buildvcs=false ./...

build:
	go build -buildvcs=false -o bin/server ./cmd/server
	go build -buildvcs=false -o bin/migrate ./cmd/migrate

generate:
	sqlc generate
	go tool gqlgen generate
