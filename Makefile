all: test install run

install:
	GOBIN=$(GOPATH)/bin GO15VENDOREXPERIMENT=1 go install worklog.go

glide:
	go get github.com/Masterminds/glide

test: glide
	GO15VENDOREXPERIMENT=1 go test -cover `glide novendor`

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
