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
docker run -d -p 7676:7676 bobrik/marathoner-updater:1.10 \
  -m http://marathon1:8080,http://marathon2:8080 -i 1
```

### Listener

The following command runs marathoner listener with
two updaters and publishes apps on `127.0.0.1`.
Notice that you need to run listener with `--net=host`.

```
docker run -d --net=host bobrik/marathoner-listener:1.10 \
  -u marathoner-updater1:7676,marathoner-updater2:7676 -b 127.0.0.1
```

### Logger

The following command runs marathoner logger with
specified updater and logs state changes to stdout:

```
docker run --rm bobrik/marathoner-logger:1.10 -u marathoner-updater1:7676
```

### Exposing apps

Marathon apps that needs to be exported should have label
`marathoner_haproxy_enabled` set to `true`.

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

## Version history

* 1.10
  * Updater: only asking for apps with marathoner enabled
* 1.9
  * Updater: 20 second timeout for getting state from marathon
* 1.8
  * Listener: Configuring timeout to kill previous haproxy instances
* 1.7
  * Listener: Fixed backports repo for haproxy
  * Listener: Haproxy is launched and managed entirely from listener
  * Listener: Constant retry in listener loop
  * Listener: Externalized haproxy template
  * Listener: Added labels to HaproxyApp structure
  * Listener: Only apps with marathoner_haproxy_enabled label set to true appear in haproxy
* 1.6
  * Higher timeouts for idle connections, more connections (only listener)
* 1.5
  * Removed port range support since it required coordination
  * Logging current updater endpoint in listener and logger
  * Worakround for https://github.com/mesosphere/marathon/issues/1106
* 1.4
  * Added labels for apps
  * Using `/v2/apps?embed=apps.tasks` instead of `/v2/tasks`
  * Using `startedAt` from marathon
  * Depends on unreleased yet marathon 0.8.0
  * Port range support
  * golint fixes
* 1.3
  * seeding random generator for better load distribution
* 1.2
  * added logger container
* 1.1
  * updater: avoid adding tasks that are not started
* 1.0
  * first public version

## TODO

* tagged automated builds with godep maybe
* exit trap in listener for faster exit
