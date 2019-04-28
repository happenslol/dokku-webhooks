include ../../common.mk

GO_ARGS ?= -a

SUBCOMMANDS = subcommands/create subcommands/delete subcommands/disable subcommands/enable subcommands/listen subcommands/logs subcommands/secret subcommands/set-secret subcommands/stop subcommands/trigger
build-in-docker: clean
	docker run --rm \
		-v $$PWD/../..:$(GO_REPO_ROOT) \
		-w $(GO_REPO_ROOT)/plugins/webhooks \
		$(BUILD_IMAGE) \
		bash -c "GO111MODULE=on GO_ARGS='$(GO_ARGS)' make -j4 build" || exit $$?

build: commands subcommands build-server

commands: **/**/commands.go
	go build $(GO_ARGS) -o commands src/commands/commands.go

subcommands: $(SUBCOMMANDS)

build-server:
	mkdir server-app && \
	go build $(GO_ARGS) -o server-app/server server/*.go && \
	cp server/Dockerfile server-app && \
	cp server/Procfile server-app && \
	cp server/CHECKS server-app

subcommands/%: src/subcommands/*/%.go
	go build $(GO_ARGS) -o $@ $<

clean:
	rm -rf commands subcommands server-app

src-clean:
	rm -rf .gitignore src vendor server Makefile *.go
