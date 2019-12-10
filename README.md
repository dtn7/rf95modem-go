# rf95modem-go

[![GoDoc](https://godoc.org/github.com/dtn7/rf95modem-go/rf95?status.svg)][godoc]
[![Build Status](https://travis-ci.org/dtn7/rf95modem-go.svg?branch=master)][travis]

Golang library to send and receive data over LoRa via a serial connection to
a [rf95modem].

This library was tested against the rf95modem commit [`acf3721`][rf95modem-commit],
informally version 0.5.1.


## Library

The primary focus of this library is the sending and receiving of data via
LoRa. An `io.Reader` and `io.Writer` are provided for this purpose. Furthermore,
both frequency and modem mode can be changed. For more information take a look
at the [documentation][godoc].

```go
package main

import (
	"fmt"

	"github.com/dtn7/rf95modem-go/rf95"
)

func main() {
	modem, modemErr := rf95.OpenModem("/dev/ttyUSB0")
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


##  Example: rf95pty

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

$ st -l /dev/pts/7
```


[godoc]: https://godoc.org/github.com/dtn7/rf95modem-go/rf95
[rf95modem]: https://github.com/gh0st42/rf95modem 
[rf95modem-commit]: https://github.com/gh0st42/rf95modem/commit/acf3721b571fa44e12d53ec77f17e2367e898e65
[travis]: https://travis-ci.org/dtn7/rf95modem-go
