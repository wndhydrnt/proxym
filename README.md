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

It takes a modular approach by dividing responsibility into three parts:
[Notifiers](http://godoc.org/github.com/wndhydrnt/proxym/types#Notifier),
[ServiceGenerators](http://godoc.org/github.com/wndhydrnt/proxym/types#ServiceGenerator) and
[ConfigGenerators](http://godoc.org/github.com/wndhydrnt/proxym/types#ConfigGenerator).

## Modules

### File

Defines a ServiceGenerator that reads configuration from files.
Configuration files are written in JSON.

### HAProxy

Takes [Service](http://godoc.org/github.com/wndhydrnt/proxym/types#Service)s and writes the configuration file HAProxy,
restarting it in case the configuration has changed.

### Marathon

Provides a Notifier that registers a callback with the [event bus](https://mesosphere.github.io/marathon/docs/event-bus.html)
of Marathon and triggers a refresh whenever it receives a `status_update_event`.

A ServiceGenerator queries Marathon for [applications](https://mesosphere.github.io/marathon/docs/rest-api.html#get-/v2/apps) and
[tasks](https://mesosphere.github.io/marathon/docs/rest-api.html#get-/v2/tasks).

### Signal

Triggers a refresh whenever the process receives a `SIGUSR1` signal. The signal can be send by software such as Ansible,
Chef or Puppet.

## Logging

The [log](./log/log.go) package defines the loggers `AppLog`, which writes to `STDOUT`, and `ErrorLog`, which writes
`STDERR`.

The level of `AppLog` is configurable while the level of `ErrorLog` is `ERROR`.

Environment variables:

Name | Required | Default
---- + -------- + -------
PROXYM_LOG_APPLOG_LEVEL | no | `INFO`
PROXYM_LOG_FORMAT | no | `%{time:02.01.2006 15:04:05} [%{level}] %{longfunc}: %{message}`

All available format options can be found in the [docs](http://godoc.org/github.com/op/go-logging#NewStringFormatter)
of `go-logging`.
