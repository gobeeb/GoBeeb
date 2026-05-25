.PHONY: fmt vet lint test bench cover

GO ?= go
PKGS ?= ./...
COVER_OUT ?= cover.out

fmt:
	$(GO) fmt $(PKGS)

vet:
	$(GO) vet $(PKGS)

lint:
	golangci-lint run

test:
	$(GO) test -race $(PKGS)

bench:
	$(GO) test -bench=. -benchmem -benchtime=5s ./mos6502/

cover:
	$(GO) test -coverprofile=$(COVER_OUT) -covermode=atomic $(PKGS)
	$(GO) tool cover -func=$(COVER_OUT) | tail -1
