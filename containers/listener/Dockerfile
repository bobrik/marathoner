FROM golang:1.4-wheezy

RUN echo "deb http://http.debian.net/debian wheezy-backports main" >> /etc/apt/sources.list.d/backports.list && \
    apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends haproxy

ADD ./haproxy.cfg.template /etc/haproxy/haproxy.cfg.template
ADD ./run.sh /run.sh

ADD ./src/ /go/src
RUN GOPATH=/go go get github.com/bobrik/marathoner/...

ENTRYPOINT ["/run.sh"]
