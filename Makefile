.PHONY: up down infra-up infra-down logs test-go test-python test build

infra-up:
	docker-compose up -d postgres redis redpanda neo4j minio redpanda-init minio-init

infra-down:
	docker-compose stop postgres redis redpanda neo4j minio redpanda-init minio-init

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

test-go:
	cd services/api-gateway && go test ./...
	cd services/ingestor && go test ./...
	cd services/event-worker && go test ./...

test-python:
	cd services/agent-service && python -m pytest tests/ -v

test: test-go test-python

build:
	docker-compose build
