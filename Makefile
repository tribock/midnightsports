IMAGE ?= midnightsports:latest

.PHONY: run down db db-reset dev test build

## run: App und PostgreSQL in Docker starten (http://localhost:8080)
run:
	docker compose up --build

## down: Alle Container stoppen
down:
	docker compose down

## db: Nur PostgreSQL starten
db:
	docker compose up -d --wait postgres

## db-reset: Datenbank löschen und neu anlegen
db-reset:
	docker compose down -v
	$(MAKE) db

## dev: Backend und Frontend mit Hot Reload starten (http://localhost:5173)
dev: db web/node_modules
	@trap 'kill 0' INT TERM EXIT; \
	(cd api && go run ./cmd/server) & \
	(cd web && npm run dev) & \
	wait

## test: Tests für Backend und Frontend ausführen
test: web/node_modules
	cd api && go vet ./... && go test ./...
	cd web && npm test

## build: Docker-Image bauen
build:
	docker build -f build/Dockerfile -t $(IMAGE) .

web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules
