hello:
	echo "Hello"

build:
	go build -o bin/main main.go

test:
	go test ./...

check:
	go vet ./...

lint:
	golangci-lint
	
run:
	go run main.go

all: hello build