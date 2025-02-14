install:
	go mod download

run:
	CGO_ENABLED=1 GOOS=linux go run main.go

build:
	CGO_ENABLED=1 GOOS=linux go build -a -o main main.go

test:
	mkdir -p ./tests && go test -v ./... && go test -v ./... -coverprofile=./tests/coverage.out -json ./... > ./tests/report.json

cleandep:
	go mod tidy
