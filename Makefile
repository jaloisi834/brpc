build:
	docker-compose build

start:
	docker-compose up -d --remove-orphans

stop:
	docker-compose down

test:
	go list ./... | grep -v /vendor/ | xargs go test
