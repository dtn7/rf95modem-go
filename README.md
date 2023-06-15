# rf95modem-go

[![GoDoc](https://godoc.org/github.com/dtn7/rf95modem-go/rf95?status.svg)][godoc]
[![Build Status](https://travis-ci.org/dtn7/rf95modem-go.svg?branch=master)][travis]

Golang library to send and receive data over LoRa via a serial connection to
a [rf95modem].

This library was tested against the rf95modem commit [`117878a`][rf95modem-commit],
slightly after version 0.7.3 including [patch #13][rf95modem-pr13].


## Library

The primary focus of this library is the sending and receiving of data via
LoRa. An `io.Reader` and `io.Writer` are provided for this purpose. It is also
possible to register handler functions for incoming messages. Furthermore, both
frequency and modem mode can be changed. For more information take a look at
the [documentation][godoc].

```go
package main

import (
	"fmt"

	"github.com/dtn7/rf95modem-go/rf95"
)

func main() {
	modem, modemErr := rf95.OpenSerial("/dev/ttyUSB0")
	if modemErr != nil {
		panic(modemErr)
	}

	if _, err := modem.Write([]byte("hello world")); err != nil {
		panic(err)
	}

	buf := make([]byte, 64)
	if _, err := modem.Read(buf); err != nil {
		panic(err)
	} else {
		fmt.Printf("%x\n", buf)
	}

	if err := modem.Close(); err != nil {
		panic(err)
	}
}
```


## Example: rf95logger

A simple logger script for incoming messages with their RSSI and SNR.

```
$ go build ./cmd/rf95logger
```

```
# Logging messages from /dev/ttyUSB0 at 868.1 MHz on mode 1, fast+short range
$ ./rf95logger /dev/ttyUSB0 868.1 1 | tee loralog.csv
```


## Example: rf95pty

A small proof of concept is `rf95pty` to bind a [rf95modem] to a new pseudoterminal
device. This code should work for POSIX operating systems.

```
$ go build ./cmd/rf95pty
```

```
# Node A provides a shell over LoRa - stupid idea, btw

$ ./rf95pty /dev/ttyUSB0
Starting modem with Status(...)
Opening pty device /dev/pts/5

$ socat /dev/pts/5,raw,nonblock,echo=0 exec:sh,pty,stderr,setsid,sigint,sane
```

```
# Node B uses this shell

$ ./rf95pty /dev/ttyUSB1
Starting modem with Status(...)
Opening pty device /dev/pts/7

$ screen /dev/pts/7
```


[godoc]: https://godoc.org/github.com/dtn7/rf95modem-go/rf95
[rf95modem]: https://github.com/gh0st42/rf95modem 
[rf95modem-commit]: https://github.com/gh0st42/rf95modem/commit/117878a4b609f9488ad8d5176f98067b9e8baa01
[rf95modem-pr13]: https://github.com/gh0st42/rf95modem/pull/16
[travis]: https://travis-ci.org/dtn7/rf95modem-go
