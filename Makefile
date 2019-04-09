dev-server:
	go run ./cmd/ghost-host/main.go

test:
	go list ./... | grep -v /vendor/ | xargs go test
