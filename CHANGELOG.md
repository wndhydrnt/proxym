# Changelog

## 1.6.0 (unreleased)

Features:
* [STDOUT] Add module stdout

Improvements:
* [Marathon] Configure protocol, domains and config through labels

Bug Fixes:
* [Annotation API] proxym does not exit in case the ZK con is lost [#17](https://github.com/wndhydrnt/proxym/issues/17)
* [Docs] Fix wrong link to `manager.RegisterHttpHandler`

## 1.5.0

Features:
* [Hipache] Add support for Hipache

Improvements:

* [Manager] Wrap each HTTP handler with a Prometheus handler to expose metrics
* [Marathon] Support applications that use the `HOST` network
* [Proxy] Define a command to check the configuration file

Bug Fixes:
* [Docs] Syntax error in example configuration file of HAProxy
* [Marathon] Fix ServiceGenerator only picks the first server in the list [#16](https://github.com/wndhydrnt/proxym/issues/16)
* [Mesos Master] Fix panic when no master could be parsed [#19](https://github.com/wndhydrnt/proxym/issues/19)

## 1.4.0

Features:

* [Core] Support proxying based on path

Improvements:

* [Manager] `manager.RegisterHttpEndpoint` expects only a path and not a prefix + pattern
* [Manager] Use one metric with different label names to record successful and failed runs
* [Mesos Master] Reduce number of requests to a Mesos Master

Bug Fixes:
* [Mesos Master] Do not prefix the Id of emitted service with `/`

## 1.3.0

Features:

* Proxy Config Generator: Generate the configuration file from a template.
  Uses the hugo templating engine.
* `types.Service` now exposes a `Source` field which can be checked in templates.
* Added the ability to define `ApplicationProtocol` and `TransportProtocol` in `types.Service`.
* New module Annotation Api
* Notifier in module File that triggers when a file is added/changed/removed.
* Expose metrics of processed and errored runs

Improvements:

* Make HAProxy module work with other load balancers. Rename it to Proxy.
* Proxy Config Generator: Do not detect HTTP via the `ServicePort` of a service.

Bugfixes:

* Marathon Service Generator: Fix error when retrieving `/v2/tasks` from Marathon 0.8.1.

## 1.2.2

Bugfixes:

* Fix panic due to Marathon ServiceGenerator processing tasks without any ports

## 1.2.1

Bugfixes:

* Configured domain not recognized in mesos_master module

## 1.2.0

Features:

* Introduce application and error loggers
* Mesos Master module

Improvements:

* Trigger a refresh right after startup
* Simplified Upstart script

## 1.1.0

Improvements:

* Assign additional domains to a service in the config of HAProxy

## 1.0.1

Features:

* Create debian package during release process

Improvements:

* Align usage of environment variables

## 1.0.0

Initial release
