# Custom WebSocket implementation library

![ci](https://github.com/Mort4lis/websocket/actions/workflows/main.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/Mort4lis/websocket.svg)](https://pkg.go.dev/github.com/Mort4lis/websocket)
![go-version](https://img.shields.io/github/go-mod/go-version/Mort4lis/websocket)
![code-size](https://img.shields.io/github/languages/code-size/Mort4lis/websocket)
![total-lines](https://img.shields.io/tokei/lines/github/Mort4lis/websocket)

![Alt text](./images/websockets-golang.png)

## Motivation

The main purpose of this developed package is education. I hold the rule which means if you want to figure out or
understand something, you should to try to implement it. In the process of implementation I kept to
[this article](https://yalantis.com/blog/how-to-build-websockets-in-go/)
and [RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455).

## Description

This package is a custom WebSocket implementation library. It includes set of types, functions and methods using which
you can easily create both client-side and server-side applications which can communicate on the WebSocket protocol.

### Features that have already been done:

* ✅ Framing
* ✅ Pings/Pongs
* ✅ Reserved Bits
* ✅ Opcodes
* ✅ Fragmentation
* ✅ UTF-8 Handling
* ✅ Limits/Performance
* ✅ Opening and Closing Handshake

### What's not done:

* ❌ Compression

## Testing

For testing package I used [Autobahn library](https://github.com/crossbario/autobahn-testsuite). If you want to look at
the Autobahn's report you should clone this repository and run test suites
(`make test`) being in the folder with the project.

## Installation

```bash
$ go get github.com/Mort4lis/websocket
```

## Usage

The `websocket.Conn` type represents the WebSocket connection. if you are developing a server application you should
use `websocket.Upgrade` function in your http handler to switching protocol to WebSocket.

```go
package main

import (
	"log"
	"net/http"

	"github.com/Mort4lis/websocket"
)

func handler(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Upgrade(w, req)
	if err != nil {
		log.Println(err)
		return
	}

	_ = conn
}
```

Otherwise, if you are interesting to use websocket package as a client you should invoke `Dial` at first (
or `DialContext`). For example:

```go
package main

import (
	"log"
	"time"

	"github.com/Mort4lis/websocket"
)

func main() {
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, err := dialer.Dial("ws://localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	_ = conn
}
```

Having `Conn` instance you can send and receive message due WebSocket protocol, calling
`WriteMessage` and `ReadMessage`.

```go
package main

import "github.com/Mort4lis/websocket"

func Echo(conn *websocket.Conn) error {
	typ, payload, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	if err = conn.WriteMessage(typ, payload); err != nil {
		return err
	}

	return nil
}
```

For more detailed information please visit [documentation](https://pkg.go.dev/github.com/Mort4lis/websocket).