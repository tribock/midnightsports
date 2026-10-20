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

# Google Cloud (Demo und Produktion), können in .env überschrieben werden
GCP_PROJECT_ID ?= midnightsports
GCP_REGION ?= europe-west6

.PHONY: run down db db-reset dev test build version demo-yaml deploy-demo deploy-yaml deploy instances logos

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

# Lädt .env und setzt die gemeinsamen Variablen für die Vorlagen in build/deploy/.
# require prüft, dass Variablen gesetzt sind.
define DEPLOY_ENV
set -a; [ -f .env ] && . ./.env; set +a; \
export GCP_PROJECT_ID="$${GCP_PROJECT_ID:-$(GCP_PROJECT_ID)}"; \
export GCP_REGION="$${GCP_REGION:-$(GCP_REGION)}"; \
export IMAGE_TAG="$(VERSION)"; \
export RUN_IMAGE="$$GCP_REGION-docker.pkg.dev/$$GCP_PROJECT_ID/dockerhub/$(patsubst docker.io/%,%,$(IMAGE))"; \
export DEPLOY_LABEL="$$(date -u +%Y%m%d-%H%M%S)"; \
export GCP_PROJECT_NUMBER="$${GCP_PROJECT_NUMBER:-$$(gcloud projects describe $$GCP_PROJECT_ID --format='value(projectNumber)')}"; \
oauth_client_id="$$(jq -r '.web.client_id // empty' secrets/midnightsports.json 2>/dev/null)"; \
require() { for v; do [ -n "$${!v}" ] || { echo "Fehlt: $$v (in .env setzen)" >&2; exit 1; }; done; }
endef

# Variablen für build/deploy/demo.yaml
define DEMO_ENV
$(DEPLOY_ENV); \
export DEMO_GOOGLE_CLIENT_ID="$${DEMO_GOOGLE_CLIENT_ID:-$$oauth_client_id}"; \
require GCP_PROJECT_NUMBER DEMO_GOOGLE_CLIENT_ID ADMIN_EMAIL GOOGLE_WALLET_ISSUER_ID
endef

# Variablen für build/deploy/deploy.yaml (Produktion). PROD_URL: eigene Domain, sonst die Adresse von Cloud Run
define PROD_ENV
$(DEPLOY_ENV); \
export SERVICE="$${SERVICE:-midnightsports}"; \
export DB_INSTANCE="$${DB_INSTANCE:-midnightsports-db}"; \
export PROD_GOOGLE_CLIENT_ID="$${PROD_GOOGLE_CLIENT_ID:-$$oauth_client_id}"; \
export PROD_URL="$${PROD_URL:-https://$$SERVICE-$$GCP_PROJECT_NUMBER.$$GCP_REGION.run.app}"; \
require GCP_PROJECT_NUMBER PROD_GOOGLE_CLIENT_ID ADMIN_EMAIL GOOGLE_WALLET_ISSUER_ID
endef

# Nur diese Platzhalter ersetzen, alle anderen $-Zeichen bleiben unverändert
DEPLOY_SUBST := $$GCP_PROJECT_ID $$GCP_REGION $$GCP_PROJECT_NUMBER $$RUN_IMAGE $$IMAGE_TAG $$DEPLOY_LABEL $$ADMIN_EMAIL $$GOOGLE_WALLET_ISSUER_ID $$APPLE_TEAM_ID
DEMO_SUBST := $(DEPLOY_SUBST) $$DEMO_GOOGLE_CLIENT_ID $$DEMO_APPLE_LOGIN_CLIENT_ID $$DEMO_APPLE_LOGIN_KEY_ID
PROD_SUBST := $(DEPLOY_SUBST) $$SERVICE $$DB_INSTANCE $$PROD_URL $$PROD_GOOGLE_CLIENT_ID $$APPLE_PASS_TYPE_ID $$PROD_APPLE_LOGIN_CLIENT_ID $$PROD_APPLE_LOGIN_KEY_ID

# Ersetzt die Platzhalter in $(1) mit $(2), deployt Service $(3) und prüft, dass er erreichbar ist.
# services replace setzt keine Berechtigungen: ohne allUsers als Invoker antwortet Cloud Run mit 403
define REPLACE_SERVICE
dir=$$(mktemp -d); trap 'rm -rf "$$dir"' EXIT; \
envsubst '$(2)' < $(1) > "$$dir/service.yaml"; \
echo "Deploy $$RUN_IMAGE:$$IMAGE_TAG ($$GCP_PROJECT_ID, $$GCP_REGION)"; \
gcloud run services replace "$$dir/service.yaml" --region="$$GCP_REGION" --project="$$GCP_PROJECT_ID" && \
gcloud run services add-iam-policy-binding "$(3)" --region="$$GCP_REGION" --project="$$GCP_PROJECT_ID" \
	--member=allUsers --role=roles/run.invoker --quiet > /dev/null && \
url=$$(gcloud run services describe "$(3)" --region="$$GCP_REGION" --project="$$GCP_PROJECT_ID" --format='value(status.url)') && \
for i in 1 2 3 4 5 6; do \
	status=$$(curl -s -o /dev/null -w '%{http_code}' "$$url/api/health"); \
	[ "$$status" = 200 ] && { echo "Online: $$url ($$(curl -s "$$url/api/version"))"; exit 0; }; \
	sleep 5; \
done; \
echo "Nicht erreichbar: $$url/api/health antwortet mit $$status" >&2; exit 1
endef

## demo-yaml: Demo-Konfiguration mit Werten aus .env ausgeben (zur Kontrolle)
demo-yaml:
	@$(DEMO_ENV); envsubst '$(DEMO_SUBST)' < build/deploy/demo.yaml

## deploy-demo: Image bauen und Demo auf Cloud Run deployen
deploy-demo: build
	@$(DEMO_ENV); $(call REPLACE_SERVICE,build/deploy/demo.yaml,$(DEMO_SUBST),midnightsports-demo)

## deploy-yaml: Produktions-Konfiguration mit Werten aus .env ausgeben (zur Kontrolle)
deploy-yaml:
	@$(PROD_ENV); envsubst '$(PROD_SUBST)' < build/deploy/deploy.yaml

## deploy: Image bauen und Produktion auf Cloud Run deployen
deploy: build
	@$(PROD_ENV); $(call REPLACE_SERVICE,build/deploy/deploy.yaml,$(PROD_SUBST),$$SERVICE)

## logos: Logos und Icons aus docs/logo/*.eps erzeugen (braucht python3 und rsvg-convert, brew install librsvg)
logos:
	python3 build/logo/generate.py

## instances: Laufende Instanzen pro Revision der letzten 5 Minuten (SERVICE=midnightsports-demo für die Demo)
instances:
	@set -a; [ -f .env ] && . ./.env; set +a; \
	project="$${GCP_PROJECT_ID:-$(GCP_PROJECT_ID)}"; service="$${SERVICE:-midnightsports}"; \
	end=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	start=$$(date -u -v-5M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '-5 min' +%Y-%m-%dT%H:%M:%SZ); \
	curl -sf -G -H "Authorization: Bearer $$(gcloud auth print-access-token)" \
		"https://monitoring.googleapis.com/v3/projects/$$project/timeSeries" \
		--data-urlencode "filter=metric.type=\"run.googleapis.com/container/instance_count\" AND resource.labels.service_name=\"$$service\"" \
		--data-urlencode "interval.startTime=$$start" --data-urlencode "interval.endTime=$$end" \
	| jq -r --arg s "$$service" '[.timeSeries[]? | {rev: .resource.labels.revision_name, state: .metric.labels.state, n: (.points[0].value.int64Value | tonumber)}] \
		| "\($$s): \(map(.n) | add // 0) Instanz(en), Messwerte bis 3 Minuten verzögert", \
		(.[] | select(.n > 0) | "  \(.rev) \(.state): \(.n)")'

web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules
