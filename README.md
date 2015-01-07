# Marathoner

Marathoner is a service discovery for marathon. It publishes
service ports from marathon on desired interface so you always
talk to `127.0.0.1` and known port. It manages haproxy and
ensures that configuration is updated as soon as there are
changes in running marathon tasks.

Marathoner consists of two parts:

* Listeners listen for updates and handles config update and haproxy reload.
* Updaters keep track of running and healthy apps and pushes changes to listeners.

You would probably want to run 3-5 updaters (one per marathon server)
and listeners on each node for high availability.

You can also run optional logger to log every state change that happens.


## Ascii view

```

        +---------+   +---------+   +---------+
        |marathon1|   |marathon2|   |marathon3|
        +----^----+   +----^----+   +----^----+
             |             |             |
             |             |             |
             ^-------------^-------------^  polling random alive marathon
             |             |             |
         +---+---+     +---+---+     +---+---+
         |updater|     |updater|     |updater|
         +---+---+     +---+---+     +---+---+
             |             |             |
             |             |             |  pushing to every connected client
             |             |             |
       +-----v------+------v------+------v------+
       |            |             |             |
       |            |             |             |
  +----v----+  +----v----+   +----v----+   +----v----+
  |listener1|  |listener1|   |listener1|   |listener1|
  +----+----+  +----+----+   +----+----+   +----+----+
       |            |             |             |
       |            |             |             |  reloading local haproxy
       |            |             |             |
   +---v---+    +---v---+     +---v---+     +---v---+
   |haproxy|    |haproxy|     |haproxy|     |haproxy|
   +---^---+    +---^---+     +---^---+     +---^---+
       |            |             |             |
       |            |             |             |  apps talk to local haproxy
       |            |             |             |
     +-+-+        +-+-+         +-+-+         +-+-+
     |app|        |app|         |app|         |app|
     +---+        +---+         +---+         +---+

```

## Features

* Nearly zero configuration, everything needed is in marathon.
* Both components are independent and share nothing for high availability.
* Marathon api is only used by updaters, thus reducing load on marathon.
* Health check awareness, only healthy tasks are added to haproxy.
* Nothing breaks if marathon goes down for some reason.
* Listeners do nothing if no changes happen in marathon, no polling involved.
* Graceful restarts and config checking for haproxy to avoid downtime.
* Services available externally and internally without any app code changes.

## Usage

Marathoner is designed to run in docker containers.

### Updater

The following command runs marathoner updater with
update interval of 1 second and two marathon hosts.
Updater is reachable at port 7676.

```
docker run -d -p 7676:7676 bobrik/marathoner-updater:1.0 \
  -m http://marathon1:8080,http://marathon2:8080 -i 1
```

### Listener

The following command runs marathoner listener with
two updaters and publishes apps on `127.0.0.1`.
Notice that you need to run listener with `--net=host`.

```
docker run -d --net=host bobrik/marathoner-listener:1.0 \
  -u marathoner-updater1:7676,marathoner-updater2:7676 -b 127.0.0.1
```

### Logger

The following command runs marathoner logger with
specified updater and logs state changes to stdout:

```
docker run --rm bobrik/marathoner-logger:1.2 -u marathoner-updater1:7676
```

## Building

If you made some changes and wish to check how they work, `./containers/make.sh`
could help you with building containers. Just run:

```sh
# build listener image with the name my-listener
./containers/make.sh listener my-listener

# build updater image with the name my-updater
./containers/make.sh updater my-updater

# build logger image with the name my-logger
./containers/make.sh updater my-logger
```

## Why not confd?

Confd requires backend to store data and supports etcd and consul.
Both of them cannot survive without the leader. That setup also
needs some glue code to push marathon changes into confd backed.
That glue code needs to be highly available and avoid extra
updates on backend at the same time. That introduces too many
moving parts for now.

## Version history

* 1.2
  * added logger container
* 1.1
  * updater: avoid adding tasks that are not started
* 1.0
  * first public version

## TODO

* blended restart policy: graceful + limit on number of running old versions
* tagged automated builds with godep maybe
* external haproxy config template
