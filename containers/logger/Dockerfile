FROM golang:1.4-wheezy

ADD ./src/ /go/src
RUN GOPATH=/go go get github.com/bobrik/marathoner/...

ENTRYPOINT ["/go/bin/logger"]
