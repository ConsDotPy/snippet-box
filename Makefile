# Include the .env file
include .env
export

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	git diff --exit-code


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go clean -testcache
	go test -v -race -buildvcs -failfast ./...

## test/short: run non-short tests
.PHONY: test/short
test/short:
	go clean -testcache
	go test -v -race -buildvcs -failfast -short ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go clean -testcache
	go test -v -race -buildvcs -failfast -covermode=atomic -coverprofile=${TMP_FOLDER}/coverage.out ./...
	go tool cover -func=${TMP_FOLDER}/coverage.out
	go tool cover -html=${TMP_FOLDER}/coverage.out


## build: build the application
.PHONY: build
build:
	# Include additional build steps, like TypeScript, SCSS or Tailwind compilation here...
	go build -o=${TMP_FOLDER}/bin/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

## run: run the  application
.PHONY: run
run: build
	${TMP_FOLDER}/bin/${BINARY_NAME} -addr=${SB_ADDR} -static-dir=${SB_STATIC_DIR} -dsn=${DB_DSN}

## run/logs: run the  application
.PHONY: run/logs
run/logs: build
	${TMP_FOLDER}/bin/${BINARY_NAME} -addr=${SB_ADDR} -static-dir=${SB_STATIC_DIR} -dsn=${DB_DSN} >>${TMP_FOLDER}/info.log 2>>${TMP_FOLDER}/error.log

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live:
	go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "${TMP_FOLDER}/bin/${BINARY_NAME}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--misc.clean_on_exit "true"


# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## push: push changes to the remote Git repository
.PHONY: push
push: tidy audit no-dirty
	git push

## production/deploy: deploy the application to production
.PHONY: production/deploy
production/deploy: confirm tidy audit no-dirty
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=${TMP_FOLDER}/bin/linux_amd64/${BINARY_NAME} ${MAIN_PACKAGE_PATH}
	upx -5 ${TMP_FOLDER}/bin/linux_amd64/${BINARY_NAME}
	${TMP_FOLDER}/bin/linux_amd64/${BINARY_NAME} -addr=${SB_ADDR} -static-dir=${SB_STATIC_DIR} -dsn=${DB_DSN}
	# Include additional deployment steps here...