# proxym

`proxym` is short for proxy manager.
It generates configuration file(s) of a reverse proxy whenever something inside your system changes.

## Purpose

The original purpose of `proxym` is to use HAProxy as a reverse proxy for applications managed by
[Marathon](https://github.com/mesosphere/marathon) and running on [Apache Mesos](http://mesos.apache.org/) without any
downtime in case of a deployment.

Note that this is not so much about service discovery inside a Mesos cluster but about connecting services running in
the cluster to the outside world.

However, its design makes it easy to integrate with systems other than [Marathon](https://github.com/mesosphere/marathon).

## Design

The `proxym` executable itself is meant to run alongside a process of a reverse proxy (e.g. HAProxy), reloading its
configuration whenever it recognizes a change in an outside system.

It takes a modular approach by dividing responsibility into three parts: [Notifiers](), [ServiceGenerators]() and
[ConfigGenerators]().

## Modules

### Notifiers

#### Marathon

Registers a callback with the [event bus](https://mesosphere.github.io/marathon/docs/event-bus.html) of Marathon and
triggers a refresh whenever it receives a `status_update_event`.

#### Signal

Triggers a refresh whenever the process receives a `SIGHUP` signal. The signal can be send by software such as Ansible,
Chef or Puppet.

### Service generators

#### File

Reads configuration from files.

#### Marathon

Creates services by querying Marathon for [applications](https://mesosphere.github.io/marathon/docs/rest-api.html#get-/v2/apps) and
[tasks](https://mesosphere.github.io/marathon/docs/rest-api.html#get-/v2/tasks).

### Config Generators

#### HAProxy

Takes services and writes the configuration file HAProxy, restarting it in case the configuration has changed.
