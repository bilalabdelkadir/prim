.PHONY: test build lint studio

test:
	go test ./... -v -race

build:
	go build -o bin/prim ./cmd/prim

lint:
	golangci-lint run

studio:
	cd studio-ui && npm run dev
