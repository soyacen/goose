.PHONY: build
build:
go build -ldflags -X main.buildstamp `date '+%Y-%m-%d_%I:%M:%S'` -X main.githash `git rev-parse HEAD` cmd/protoc-gen-goose/main.go


.PHONY: install
install:
	go install ./cmd/protoc-gen-goose

.PHONY: test
test:
	go test -v ./...

.PHONY: example
example:
	protoc \
	--proto_path=. \
	--proto_path=./third_party \
	--proto_path=./../ \
	--go_out=. \
	--go_opt=paths=source_relative \
	--goose_out=. \
	--goose_opt=paths=source_relative \
	example/*/*.proto

.PHONY: all
all: install example
