
commit = $(shell git rev-parse HEAD)
version = latest

ifeq ($(OS),Windows_NT)
wharf-provider-github.exe: swag
	go build -o wharf-provider-github.exe .
else
wharf-provider-github: swag
	go build -o wharf-provider-github .
endif

.PHONY: clean
clean: clean-swag clean-build

.PHONY: clean-build
clean-build:
ifeq ($(OS),Windows_NT)
	rm -rfv wharf-provider-github.exe
else
	rm -rfv wharf-provider-github
endif

.PHONY: install
install: swag
	go install

.PHONY: check
check: swag
	go test ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: deps
deps: deps-go deps-npm

.PHONY: deps-go
deps-go:
	go install github.com/mgechev/revive@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/swaggo/swag/cmd/swag@v1.8.1
	go mod download

.PHONY: deps-npm
deps-npm:
	npm install

.PHONY: docker
docker:
	docker build . \
		--pull \
		-t "quay.io/iver-wharf/wharf-provider-github:latest" \
		-t "quay.io/iver-wharf/wharf-provider-github:$(version)" \
		--build-arg BUILD_VERSION="$(version)" \
		--build-arg BUILD_GIT_COMMIT="$(commit)" \
		--build-arg BUILD_DATE="$(shell date --iso-8601=seconds)"
	@echo ""
	@echo "Push the image by running:"
	@echo "docker push quay.io/iver-wharf/wharf-provider-github:latest"
ifneq "$(version)" "latest"
	@echo "docker push quay.io/iver-wharf/wharf-provider-github:$(version)"
endif

.PHONY: docker-run
docker-run:
	docker run --rm -it quay.io/iver-wharf/wharf-provider-github:$(version)

.PHONY: serve
serve: swag
	go run .

.PHONY: clean-swag
clean-swag:
	rm -vrf docs

.PHONY: swag-force
swag-force: clean-swag swag

.PHONY: swag
swag: docs

docs:
	swag init --parseDependency --parseDepth 2

.PHONY: lint lint-fix \
	lint-md lint-go \
	lint-fix-md lint-fix-go
lint: lint-md lint-go
lint-fix: lint-fix-md lint-fix-go

lint-md:
	npx remark . .github

lint-fix-md:
	npx remark . .github -o

lint-go:
	@echo goimports -d '**/*.go'
	@goimports -d $(shell git ls-files "*.go")
	revive -formatter stylish -config revive.toml ./...

lint-fix-go:
	@echo goimports -d -w '**/*.go'
	@goimports -d -w $(shell git ls-files "*.go")
