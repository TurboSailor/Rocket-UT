VER := 0.1.0
export VER

.PHONY: bins daemon test click deploy all

bins:
	bash scripts/fetch-bins.sh

daemon:
	cd daemon && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ../vendor-bin/rocketd .

test:
	cd daemon && go test ./...

click: daemon
	bash scripts/build-click.sh

deploy: click
	bash scripts/deploy.sh

all: bins daemon test click
