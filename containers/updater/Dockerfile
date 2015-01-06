FROM golang:1.4-wheezy

ADD ./src/ /go/src
RUN GOPATH=/go go get github.com/bobrik/marathoner/...

EXPOSE 7676

ENTRYPOINT ["/go/bin/updater"]
