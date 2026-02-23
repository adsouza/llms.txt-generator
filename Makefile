.PHONY: all frontend backend lint test clean

all: frontend backend

frontend:
	cd frontend && npm ci && npm run build

backend: frontend
	go build -o bin/server ./cmd/server

lint:
	gofmt -l .
	go vet ./...
	staticcheck ./...

test:
	go test ./...

clean:
	rm -rf bin/ static/build/ frontend/node_modules/
