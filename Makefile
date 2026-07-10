GOPATH:=$(shell go env GOPATH)

.PHONY: init
init:
	@go install github.com/golang/protobuf/protoc-gen-go@latest
	@go install github.com/asim/go-micro/cmd/protoc-gen-micro/v4@latest

.PHONY: proto
proto:
	@protoc --proto_path=. --micro_out=. --go_out=:. proto/zdai.proto
	@# Fix v4→v5 import paths in generated micro file
	@sed -i 's|go-micro.dev/v4/|go-micro.dev/v5/|g' proto/zdai.pb.micro.go
	@sed -i '/go-micro.dev\/v4\/api/d' proto/zdai.pb.micro.go

.PHONY: build
build:
	@go build -o zdai .

.PHONY: run
run:
	./zdai

.PHONY: test
test:
	@go vet ./...
	@go test ./... -cover

.PHONY: vendor
vendor:
	@go mod tidy
	@go mod vendor

.PHONY: update
update:
	@go get -u
	@go mod tidy
