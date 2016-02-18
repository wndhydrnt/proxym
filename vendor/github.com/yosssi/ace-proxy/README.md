# Ace Proxy

[![wercker status](https://app.wercker.com/status/013ef8bb0794bdc457c3f3f766677bff/m "wercker status")](https://app.wercker.com/project/bykey/013ef8bb0794bdc457c3f3f766677bff)
[![GoDoc](http://godoc.org/github.com/yosssi/ace-proxy?status.svg)](http://godoc.org/github.com/yosssi/ace-proxy)
[![Coverage Status](https://img.shields.io/coveralls/yosssi/ace-proxy.svg)](https://coveralls.io/r/yosssi/ace-proxy?branch=master)

## Overview

Ace Proxy is a proxy for the Ace template engine. This proxy caches the options for the Ace template engine so that you don't have to specify them every time calling the Ace APIs.

## Usage

```go
package main

import (
	"net/http"

	"github.com/yosssi/ace"
	"github.com/yosssi/ace-proxy"
)

var p = proxy.New(&ace.Options{
	BaseDir:       "views",
	DynamicReload: true,
})

func handler(w http.ResponseWriter, r *http.Request) {
	tpl, err := p.Load("base", "", nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tpl.Execute(w, map[string]string{"Msg": "Hello Ace"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
```
