#!/bin/sh

set -e

/go/bin/listener -p /tmp/haproxy.pid -t /etc/haproxy/haproxy.cfg.template $@
