SHELL := /bin/bash

# Version aus Git: Tag (z.B. v1.2.0) oder Commit-Hash, "-dirty" bei uncommitteten Änderungen.
# Builds mit uncommitteten Änderungen erhalten zusätzlich einen Zeitstempel, damit jeder Tag eindeutig ist.
ifndef VERSION
GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION := $(if $(findstring -dirty,$(GIT_VERSION)),$(GIT_VERSION).$(shell date -u +%Y%m%d%H%M%S),$(GIT_VERSION))
endif
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.version=$(VERSION) -X main.date=$(BUILD_DATE)
export VERSION BUILD_DATE

IMAGE ?= docker.io/tribock/midnightsports

# Google Cloud (Demo), können in .env überschrieben werden
GCP_PROJECT_ID ?= midnightsports
GCP_REGION ?= europe-west6

.PHONY: run down db db-reset dev test build version demo-yaml deploy-demo

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
	(cd api && go run -ldflags "$(LDFLAGS)" ./cmd/server) & \
	(cd web && npm run dev) & \
	wait

## test: Tests für Backend und Frontend ausführen
test: web/node_modules
	cd api && go vet ./... && go test ./...
	cd web && npm test

## version: Aktuelle Version anzeigen (wird als Image-Tag verwendet)
version:
	@echo $(VERSION)

## build: Docker-Image bauen und auf Docker Hub pushen ($(IMAGE):<version>)
build:
	docker buildx build --platform linux/arm64/v8,linux/amd64 -f build/Dockerfile \
		--build-arg VERSION=$(VERSION) --build-arg BUILD_DATE=$(BUILD_DATE) \
		--push -t $(IMAGE):$(VERSION) .

# Lädt .env und setzt alle Variablen für die Vorlage build/deploy/demo.yaml
define DEMO_ENV
set -a; [ -f .env ] && . ./.env; set +a; \
export GCP_PROJECT_ID="$${GCP_PROJECT_ID:-$(GCP_PROJECT_ID)}"; \
export GCP_REGION="$${GCP_REGION:-$(GCP_REGION)}"; \
export IMAGE_TAG="$(VERSION)"; \
export RUN_IMAGE="$$GCP_REGION-docker.pkg.dev/$$GCP_PROJECT_ID/dockerhub/$(patsubst docker.io/%,%,$(IMAGE))"; \
export DEPLOY_LABEL="$$(date -u +%Y%m%d-%H%M%S)"; \
export DEMO_GOOGLE_CLIENT_ID="$${DEMO_GOOGLE_CLIENT_ID:-$$(jq -r '.web.client_id // empty' secrets/midnightsports.json 2>/dev/null)}"; \
export GCP_PROJECT_NUMBER="$${GCP_PROJECT_NUMBER:-$$(gcloud projects describe $$GCP_PROJECT_ID --format='value(projectNumber)')}"; \
for v in GCP_PROJECT_NUMBER DEMO_GOOGLE_CLIENT_ID ADMIN_EMAIL GOOGLE_WALLET_ISSUER_ID; do \
	[ -n "$${!v}" ] || { echo "Fehlt: $$v (in .env setzen)" >&2; exit 1; }; \
done
endef

# Nur diese Platzhalter ersetzen, alle anderen $-Zeichen bleiben unverändert
DEMO_SUBST := $$GCP_PROJECT_ID $$GCP_REGION $$GCP_PROJECT_NUMBER $$RUN_IMAGE $$IMAGE_TAG $$DEPLOY_LABEL $$DEMO_GOOGLE_CLIENT_ID $$ADMIN_EMAIL $$GOOGLE_WALLET_ISSUER_ID

## demo-yaml: Demo-Konfiguration mit Werten aus .env ausgeben (zur Kontrolle)
demo-yaml:
	@$(DEMO_ENV); envsubst '$(DEMO_SUBST)' < build/deploy/demo.yaml

## deploy-demo: Image bauen und Demo auf Cloud Run deployen
deploy-demo: build
	@$(DEMO_ENV); \
	dir=$$(mktemp -d); trap 'rm -rf "$$dir"' EXIT; \
	envsubst '$(DEMO_SUBST)' < build/deploy/demo.yaml > "$$dir/demo.yaml"; \
	echo "Deploy $$RUN_IMAGE:$$IMAGE_TAG ($$GCP_PROJECT_ID, $$GCP_REGION)"; \
	gcloud run services replace "$$dir/demo.yaml" --region="$$GCP_REGION" --project="$$GCP_PROJECT_ID"

web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules
