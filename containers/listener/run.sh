#!/bin/sh

set -e

haproxy -D -f /etc/haproxy/haproxy.cfg -p /tmp/haproxy.pid

/go/bin/listener -p /tmp/haproxy.pid $@
