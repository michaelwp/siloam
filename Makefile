# Data Base Jemaat Kolom 9 — run `make help` for the list of commands.
#
#   Settings:         .env (copy of .env.example) — MONGODB_URI etc.
#   Local:            make import && make run
#   Docker:           make docker-up && make docker-import
#   Apple container:  make mac-up && make mac-import   (macOS 26+)

SHELL := /bin/bash

APP         := siloam
IMAGE       ?= siloam:latest
PORT        ?= 8080
MONGO_IMAGE ?= mongo:8
XLSX        ?= $(HOME)/Downloads/Data Base Jemaat Kolom 9 2027 - 2031.xlsx

# The app reads .env itself. Make also reads it (without exporting) so the
# container targets below can pass the same sign-in settings through.
-include .env

# .env values may be quoted ("..." or '...'); the app strips quotes, so does Make
unquote = $(patsubst '%',%,$(patsubst "%",%,$(1)))
MONGODB_URI := $(call unquote,$(MONGODB_URI))

# defaults for containers when not set in .env
WEBAUTHN_RP_ID         ?= localhost
WEBAUTHN_ORIGINS       ?= http://localhost:$(PORT)
COOKIE_SECURE          ?= false
INITIAL_ADMIN_USER     ?= admin
INITIAL_ADMIN_PASSWORD ?= admin
MAX_FAILED_LOGINS      ?= 3
LOCK_DURATION          ?= 3m
API_RATE_LIMIT         ?= 600
AUTH_RATE_LIMIT        ?= 20
TRUSTED_PROXIES        ?=
# settings passed to the app containers
APP_SETTINGS = WEBAUTHN_RP_ID WEBAUTHN_ORIGINS COOKIE_SECURE INITIAL_ADMIN_USER INITIAL_ADMIN_PASSWORD \
	MAX_FAILED_LOGINS LOCK_DURATION API_RATE_LIMIT AUTH_RATE_LIMIT TRUSTED_PROXIES
AUTH_ENV = $(foreach v,$(APP_SETTINGS),$(v)="$(call unquote,$($(v)))")

# source for `make db-copy` (the target is MONGODB_URI in .env)
FROM ?= mongodb://localhost:27017/siloam

# Apple container resources (Docker uses compose.yaml instead)
MAC_NET     := $(APP)-net
MAC_VOLUME  := $(APP)-mongo-data
MAC_MONGO   := $(APP)-mongo
MAC_APP     := $(APP)-app

# the workbook is copied here and mounted read-only into the container for import
IMPORT_DIR  := $(CURDIR)/.import

COMPOSE     := docker compose

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Local development

.PHONY: env
env: ## Create .env from .env.example (if missing)
	@test -f .env && echo ".env already exists" || (cp .env.example .env && echo "created .env — edit MONGODB_URI")

.PHONY: install
install: ## Install Go and npm dependencies
	go mod download
	cd web && npm ci

.PHONY: web
web: ## Build the React frontend into web/dist
	cd web && npm run build

.PHONY: build
build: web ## Build bin/siloam with the frontend embedded
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(APP) .

.PHONY: run
run: build ## Build and serve (default port 8080, override with PORT=); MongoDB from .env
	ADDR=:$(PORT) ./bin/$(APP)

.PHONY: dev
dev: web/dist/index.html ## Run the Go API on :8080 (pair with `make dev-web`)
	go run .

.PHONY: dev-web
dev-web: ## Run Vite with hot reload on :5173 (proxies /api to :8080)
	cd web && npm run dev

# go:embed needs web/dist to exist even when the frontend is served by Vite
web/dist/index.html:
	@mkdir -p web/dist
	@echo '<!doctype html><p>Frontend not built. Use <code>make dev-web</code> or <code>make web</code>.</p>' > $@

.PHONY: import
import: ## Import XLSX into MONGODB_URI from .env (replaces existing data)
	go run . -import "$(XLSX)" -replace

.PHONY: db-copy
db-copy: ## Copy FROM= (default local siloam) into MONGODB_URI; ONLY=families,members to skip users; REPLACE=1 to overwrite
	go run . -copy-from "$(FROM)" $(if $(ONLY),-collections $(ONLY)) $(if $(REPLACE),-replace)

.PHONY: user-reset
user-reset: ## Unlock an account and print a temporary password: make user-reset NAME=erwin
	@test -n "$(NAME)" || (echo "usage: make user-reset NAME=<username>" && exit 1)
	go run . -reset-user "$(NAME)"

.PHONY: test
test: web/dist/index.html ## Run Go tests (MongoDB tests use MONGODB_TEST_URI, default local, and skip without one)
	go test -count=1 ./...

##@ Quality

GOLANGCI_LINT ?= golangci-lint

.PHONY: lint
lint: lint-go lint-web ## Lint Go and React

.PHONY: lint-go
lint-go: web/dist/index.html ## golangci-lint (vet, staticcheck, errcheck, gofmt, goimports, …)
	@command -v $(GOLANGCI_LINT) >/dev/null || (echo "golangci-lint not found: brew install golangci-lint" && exit 1)
	$(GOLANGCI_LINT) run ./...

.PHONY: lint-web
lint-web: ## oxlint (react, hooks, a11y, typescript) + tsc type check
	cd web && npm run lint && npm run typecheck

.PHONY: lint-fix
lint-fix: ## Auto-fix lint issues and format code where possible
	$(GOLANGCI_LINT) run --fix ./...
	cd web && npm run lint:fix

.PHONY: ci
ci: lint test web ## Run what GitHub Actions runs (lint, tests, web build)

.PHONY: release
release: ## Tag and push a release, which runs CI and deploys: make release VERSION=v1.0.0
	@echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$' || (echo "usage: make release VERSION=v1.2.3" && exit 1)
	@test "$$(git branch --show-current)" = main || (echo "release from main (current: $$(git branch --show-current))" && exit 1)
	@test -z "$$(git status --porcelain)" || (echo "commit or stash your changes first" && exit 1)
	@git fetch -q origin main && test "$$(git rev-parse HEAD)" = "$$(git rev-parse origin/main)" || (echo "main is not in sync with origin/main — push or pull first" && exit 1)
	@! git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null || (echo "tag $(VERSION) already exists" && exit 1)
	git tag -a "$(VERSION)" -m "Release $(VERSION)"
	git push origin "$(VERSION)"
	@echo "→ CI: https://github.com/michaelwp/siloam/actions"

.PHONY: fmt
fmt: ## Format Go code (gofmt + goimports)
	$(GOLANGCI_LINT) fmt ./...

.PHONY: hooks
hooks: ## Install the git pre-commit hook (runs lint on staged changes)
	@git rev-parse --git-dir >/dev/null 2>&1 || git init -q
	git config core.hooksPath .githooks
	@echo "pre-commit hook installed (.githooks/pre-commit)"

.PHONY: setup
setup: install hooks ## First-time setup: dependencies + git hook

.PHONY: clean
clean: ## Remove build output
	rm -rf bin web/dist $(IMPORT_DIR)

# copy the workbook to a space-free path that can be mounted into a container
.PHONY: stage-xlsx
stage-xlsx:
	@test -f "$(XLSX)" || (echo "XLSX not found: $(XLSX)" && exit 1)
	@mkdir -p $(IMPORT_DIR)
	@cp "$(XLSX)" $(IMPORT_DIR)/data.xlsx

# Which MongoDB the containers use: MONGODB_URI from .env when it points to
# another machine (e.g. Atlas); a localhost URI is unreachable from inside a
# container, so then the bundled MongoDB container is started instead.
LOCAL_URI := $(or $(if $(MONGODB_URI),,none),$(findstring localhost,$(MONGODB_URI)),$(findstring 127.0.0.1,$(MONGODB_URI)))
REMOTE_DB := $(if $(LOCAL_URI),,1)
BUNDLED_URI_DOCKER = mongodb://mongo:27017/$(APP)
BUNDLED_URI_MAC    = mongodb://$(mac_mongo_ip):27017/$(APP)

.PHONY: db-which
db-which: ## Show which MongoDB local runs and containers will use
	@echo "local (make run/import/db-copy): $$(echo '$(MONGODB_URI)' | sed -E 's#(://[^:/@]+:)[^@]*@#\1****@#')"
	@echo "containers:                      $(if $(REMOTE_DB),same as above (from .env),bundled MongoDB container ($(BUNDLED_URI_DOCKER)))"

##@ Docker (compose: app + mongo)

.PHONY: docker-build
docker-build: ## Build the Docker image
	IMAGE=$(IMAGE) $(COMPOSE) build

.PHONY: docker-up
docker-up: ## Build and start the app (+ bundled MongoDB unless .env points elsewhere)
ifdef REMOTE_DB
	IMAGE=$(IMAGE) PORT=$(PORT) $(AUTH_ENV) CONTAINER_MONGODB_URI="$(MONGODB_URI)" $(COMPOSE) up -d --build --no-deps app
	@echo "→ http://localhost:$(PORT)  (MongoDB: MONGODB_URI from .env)"
else
	IMAGE=$(IMAGE) PORT=$(PORT) $(AUTH_ENV) $(COMPOSE) up -d --build
	@echo "→ http://localhost:$(PORT)  (MongoDB: bundled container)"
endif

.PHONY: docker-import
docker-import: stage-xlsx ## Import XLSX into the containers' MongoDB (replaces data)
	IMAGE=$(IMAGE) $(if $(REMOTE_DB),CONTAINER_MONGODB_URI="$(MONGODB_URI)") $(COMPOSE) run --rm $(if $(REMOTE_DB),--no-deps) -v $(IMPORT_DIR):/import:ro app -import /import/data.xlsx -replace

.PHONY: docker-db-copy
docker-db-copy: ## Copy the bundled Docker MongoDB into MONGODB_URI from .env; REPLACE=1 to overwrite
	@test -n "$(REMOTE_DB)" || (echo "MONGODB_URI in .env must point to the new (non-localhost) database" && exit 1)
	IMAGE=$(IMAGE) $(COMPOSE) up -d --wait mongo
	IMAGE=$(IMAGE) $(COMPOSE) run --rm --no-deps -e MONGODB_URI="$(MONGODB_URI)" app -copy-from $(BUNDLED_URI_DOCKER) $(if $(REPLACE),-replace)

.PHONY: docker-user-reset
docker-user-reset: ## Unlock an account in the containers' MongoDB, print a temporary password: make docker-user-reset NAME=erwin
	@test -n "$(NAME)" || (echo "usage: make docker-user-reset NAME=<username>" && exit 1)
	@IMAGE=$(IMAGE) $(if $(REMOTE_DB),CONTAINER_MONGODB_URI="$(MONGODB_URI)") $(COMPOSE) run --rm $(if $(REMOTE_DB),--no-deps) app -reset-user "$(NAME)"

.PHONY: docker-down
docker-down: ## Stop containers (data is kept)
	$(COMPOSE) down

.PHONY: docker-logs
docker-logs: ## Follow logs
	$(COMPOSE) logs -f

.PHONY: docker-ps
docker-ps: ## Show container status
	$(COMPOSE) ps

.PHONY: docker-mongosh
docker-mongosh: ## Open a mongosh shell on the containers' MongoDB
ifdef REMOTE_DB
	mongosh "$(MONGODB_URI)"
else
	$(COMPOSE) exec mongo mongosh $(APP)
endif

.PHONY: docker-clean
docker-clean: ## Stop containers and DELETE the bundled MongoDB volume
	$(COMPOSE) down -v --remove-orphans

##@ Apple container (macOS 26+, `container` CLI)

# IPv4 of the MongoDB container on $(MAC_NET); Apple container has no
# service-name DNS like compose, so the app is pointed at the IP directly.
mac_mongo_ip = $$(container inspect $(MAC_MONGO) | grep -oE '"(ipv4A|a)ddress" *: *"[0-9.]+' | head -1 | grep -oE '[0-9.]+$$')

.PHONY: mac-start
mac-start: ## Start the container system service (first run installs a kernel)
	@container system status >/dev/null 2>&1 || container system start --enable-kernel-install

.PHONY: mac-build
mac-build: mac-start ## Build the image with Apple container
	container build -t $(IMAGE) .

.PHONY: mac-net
mac-net: mac-start
	@container network inspect $(MAC_NET) >/dev/null 2>&1 || container network create $(MAC_NET) >/dev/null

.PHONY: mac-mongo
mac-mongo: mac-net
	@container volume inspect $(MAC_VOLUME) >/dev/null 2>&1 || container volume create $(MAC_VOLUME) >/dev/null
	@if ! container inspect $(MAC_MONGO) >/dev/null 2>&1; then \
		echo "starting $(MAC_MONGO)"; \
		container run -d --name $(MAC_MONGO) --network $(MAC_NET) -v $(MAC_VOLUME):/data/db $(MONGO_IMAGE) >/dev/null; \
	elif ! container ls | grep -q '^$(MAC_MONGO) '; then \
		container start $(MAC_MONGO) >/dev/null; \
	fi
	@echo -n "waiting for MongoDB"; \
	for i in $$(seq 1 60); do \
		container exec $(MAC_MONGO) mongosh --quiet --eval 'db.runCommand({ping:1}).ok' >/dev/null 2>&1 && echo " ok" && exit 0; \
		echo -n "."; sleep 1; \
	done; echo " timed out"; exit 1

# MongoDB URI for the app container
MAC_APP_URI = $(if $(REMOTE_DB),$(MONGODB_URI),$(BUNDLED_URI_MAC))

.PHONY: mac-up
mac-up: mac-build $(if $(REMOTE_DB),mac-net,mac-mongo) ## Build and start the app (+ bundled MongoDB unless .env points elsewhere)
	@container rm -f $(MAC_APP) >/dev/null 2>&1 || true
	container run -d --name $(MAC_APP) --network $(MAC_NET) -p $(PORT):8080 \
		-e MONGODB_URI="$(MAC_APP_URI)" \
		$(foreach v,$(APP_SETTINGS),-e $(v)="$(call unquote,$($(v)))") \
		$(IMAGE)
	@echo "→ http://localhost:$(PORT)  (MongoDB: $(if $(REMOTE_DB),MONGODB_URI from .env,bundled container))"

.PHONY: mac-import
mac-import: stage-xlsx $(if $(REMOTE_DB),mac-net,mac-mongo) ## Import XLSX into the containers' MongoDB (replaces data)
	container run --rm --network $(MAC_NET) -v $(IMPORT_DIR):/import \
		-e MONGODB_URI="$(MAC_APP_URI)" \
		$(IMAGE) -import /import/data.xlsx -replace

.PHONY: mac-db-copy
mac-db-copy: mac-mongo ## Copy the bundled Apple container MongoDB into MONGODB_URI from .env; REPLACE=1 to overwrite
	@test -n "$(REMOTE_DB)" || (echo "MONGODB_URI in .env must point to the new (non-localhost) database" && exit 1)
	container run --rm --network $(MAC_NET) -e MONGODB_URI="$(MONGODB_URI)" \
		$(IMAGE) -copy-from $(BUNDLED_URI_MAC) $(if $(REPLACE),-replace)

.PHONY: mac-user-reset
mac-user-reset: $(if $(REMOTE_DB),mac-net,mac-mongo) ## Unlock an account in the containers' MongoDB, print a temporary password: make mac-user-reset NAME=erwin
	@test -n "$(NAME)" || (echo "usage: make mac-user-reset NAME=<username>" && exit 1)
	@container run --rm --network $(MAC_NET) -e MONGODB_URI="$(MAC_APP_URI)" $(IMAGE) -reset-user "$(NAME)"

.PHONY: mac-down
mac-down: ## Stop and remove containers (data volume is kept)
	-container stop $(MAC_APP) $(MAC_MONGO) 2>/dev/null
	-container rm $(MAC_APP) $(MAC_MONGO) 2>/dev/null

.PHONY: mac-logs
mac-logs: ## Follow app logs
	container logs -f $(MAC_APP)

.PHONY: mac-ps
mac-ps: ## Show container status
	container ls -a | grep -E '^ID|$(APP)-' || true

.PHONY: mac-mongosh
mac-mongosh: ## Open a mongosh shell on the containers' MongoDB
ifdef REMOTE_DB
	mongosh "$(MONGODB_URI)"
else
	container exec -it $(MAC_MONGO) mongosh $(APP)
endif

.PHONY: mac-clean
mac-clean: mac-down ## Remove containers, network and DELETE the bundled MongoDB volume
	-container volume rm $(MAC_VOLUME) 2>/dev/null
	-container network rm $(MAC_NET) 2>/dev/null

##@ Fly.io (fly.toml)

.PHONY: fly-secrets
fly-secrets: ## Copy MONGODB_URI (and INITIAL_ADMIN_PASSWORD if set) from .env to Fly secrets
	@test -n "$(MONGODB_URI)" || (echo "MONGODB_URI is not set in .env" && exit 1)
	@{ echo 'MONGODB_URI=$(MONGODB_URI)'; $(if $(filter-out admin,$(INITIAL_ADMIN_PASSWORD)),echo 'INITIAL_ADMIN_PASSWORD=$(INITIAL_ADMIN_PASSWORD)';) } | fly secrets import --stage
	@echo "secrets staged — they take effect on the next 'make fly-deploy'"

.PHONY: fly-deploy
fly-deploy: ## Build remotely and deploy (stops if a new machine fails its health check)
	fly deploy

.PHONY: fly-status
fly-status: ## Show machines and health checks
	fly status

.PHONY: fly-logs
fly-logs: ## Show recent logs
	fly logs --no-tail

.PHONY: fly-open
fly-open: ## Open the deployed app
	fly open

##@ Android app (android/, public view + background notifications)

# JDK 17+ for Gradle: $JAVA_HOME, else a JDK unpacked in ~/.jdks
ANDROID_JAVA_HOME ?= $(or $(JAVA_HOME),$(shell ls -d $(HOME)/.jdks/jdk-*/Contents/Home 2>/dev/null | tail -1))
GRADLE = cd android && JAVA_HOME="$(ANDROID_JAVA_HOME)" ./gradlew --no-daemon -q

.PHONY: android-test
android-test: ## Run the Android app's unit tests
	$(GRADLE) testDebugUnitTest

.PHONY: android-apk
android-apk: ## Build the debug APK (android/app/build/outputs/apk/debug/app-debug.apk)
	$(GRADLE) assembleDebug
	@echo "→ android/app/build/outputs/apk/debug/app-debug.apk"

.PHONY: android-release
android-release: ## Build the release APK (unsigned; sign it before publishing)
	$(GRADLE) assembleRelease
	@echo "→ android/app/build/outputs/apk/release/app-release-unsigned.apk"

ANDROID_SDK ?= $(or $(ANDROID_HOME),$(HOME)/Library/Android/sdk)
ADB = $(or $(shell command -v adb),$(ANDROID_SDK)/platform-tools/adb)
AVD ?= siloam-test

.PHONY: android-emulator
android-emulator: ## Start the Android emulator (AVD=siloam-test) in a window and wait until it has booted
	@$(ADB) get-state >/dev/null 2>&1 && echo "a device is already connected" && exit 0; \
	nohup $(ANDROID_SDK)/emulator/emulator -avd $(AVD) -no-boot-anim >/dev/null 2>&1 & \
	echo -n "booting $(AVD)"; $(ADB) wait-for-device; \
	until [ "$$($(ADB) shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')" = 1 ]; do echo -n "."; sleep 2; done; echo " ok"

.PHONY: android-install
android-install: android-apk ## Install the debug APK on the connected phone/emulator (adb)
	@$(ADB) get-state >/dev/null 2>&1 || (echo "no phone or emulator connected: run 'make android-emulator', or plug in a phone with USB debugging on" && exit 1)
	$(ADB) install -r android/app/build/outputs/apk/debug/app-debug.apk
	$(ADB) shell am start -n id.gmimsiloam.jemaat/.MainActivity >/dev/null
