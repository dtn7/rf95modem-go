# rf95modem-go

[![GoDoc](https://godoc.org/github.com/dtn7/rf95modem-go/rf95?status.svg)](https://godoc.org/github.com/dtn7/rf95modem-go/rf95)
[![Build Status](https://travis-ci.org/dtn7/rf95modem-go.svg?branch=master)](https://travis-ci.org/dtn7/rf95modem-go)

Golang library to send and receive data over LoRa via a [rf95modem].


## Library

The primary focus of this library is the sending and receiving of data via
LoRa. An `io.Reader` and `io.Writer` are provided for this purpose.

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

A small proof of concept is `rf95pty` to bind a [rf95modem] to a pseudoterminal device.

```
$ go build ./cmd/rf95pty

# Node A: provide a shell over LoRa - stupid idea, btw
$ ./rf95pty /dev/ttyUSB0
Opening pty device /dev/pts/5
$ socat /dev/pts/5,raw,nonblock,echo=0 exec:sh,pty,stderr,setsid,sigint,sane

# Node B: Uses shell
$ ./rf95pty /dev/ttyUSB1
Opening pty device /dev/pts/7
$ st -l /dev/pts/7
```


[rf95modem]: https://github.com/gh0st42/rf95modem 
