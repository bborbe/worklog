
all: test install run

install:
	GOBIN=$(GOPATH)/bin GO15VENDOREXPERIMENT=1 go install worklog.go

test:
	go test -cover -race $(shell go list ./... | grep -v /vendor/)

format:
	go get golang.org/x/tools/cmd/goimports
	find . -type f -name '*.go' -not -path './vendor/*' -exec gofmt -w "{}" +
	find . -type f -name '*.go' -not -path './vendor/*' -exec goimports -w "{}" +

run:
	worklog \
	-author "Benjamin Borbe" \
	-dir "" \
	-logtostderr \
	-v=4

rundebug:
	worklog \
	-author "Benjamin Borbe" \
	-dir "" \
	-logtostderr \
	-v=4
