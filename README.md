# rf95modem-go

[![Go Reference](https://pkg.go.dev/badge/github.com/dtn7/rf95modem-go/rf95.svg)][godoc]
![CI](https://github.com/dtn7/rf95modem-go/workflows/CI/badge.svg)

Golang library to send and receive data over LoRa PHY via a serial connection to a [rf95modem].

This library was tested against the rf95modem commit [`117878a`][rf95modem-commit], slightly after version 0.7.3 including [patch #16][rf95modem-pr16].


## Library

The primary focus of this library is to send and receive data via LoRa's physical layer, LoRa PHY, with the help of a [rf95modem].

Therefore the `rf95.Modem` allows direct interaction with a connected rf95modem, including configuration changes, sending, and receiving raw LoRa PHY messages.
Additionally the `rf95.Stream` allows using the known `io.Reader` and `io.Writer` interfaces for data exchange.
A `rf95.Stream` is hooked to the `rf95.Modem` as a registered handler, where, of course, custom handlers can also be implemented and connected.

The following two short code examples are demonstrating how to use `rf95.Modem` and `rf95.Stream` on top.
More details are available in the [documentation][godoc].
There are also example programs available under `./cmd`, which are also described below.

```go
// Example of how to use a rf95.Modem to establish a connection, configure the
// rf95modem, send some bytes and wait for the first answere.

package main

import (
	"context"
	"fmt"

	"github.com/dtn7/rf95modem-go/rf95"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Create and configure a rf95.Modem for rf95modem usage
	modem, err := rf95.OpenSerial("/dev/ttyUSB0", context.Background())
	checkError(err)

	checkError(modem.Frequency(868.23))
	checkError(modem.Mode(rf95.FastShortRange))

	// Broadcast a message
	_, err = modem.Transmit([]byte("Hello LoRa PHY from rf95modem{,-go}"))
	checkError(err)

	// Register a handler to print a received message and exit
	finChan := make(chan struct{})

	_, err = modem.RegisterHandlers(func(rx rf95.RxMessage) {
		fmt.Printf("%#v\n", rx)
		close(finChan)
	}, nil)
	checkError(err)

	<-finChan

	checkError(modem.Close())
}
```

```go
// Another example which utilizes a rf95.Stream to send a local files and dump a
// received sample to another file.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dtn7/rf95modem-go/rf95"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Create and configure a rf95.Modem for rf95modem usage
	modem, err := rf95.OpenSerial("/dev/ttyUSB0", context.Background())
	checkError(err)

	checkError(modem.Frequency(868.23))
	checkError(modem.Mode(rf95.FastShortRange))

	// Create a rf95.Stream using our modem
	stream, err := rf95.NewStream(modem)
	checkError(err)

	// Send part of our source code over LoRa, because we are so Free(tm)
	fOut, err := os.Open("rf95/modem.go")
	checkError(err)
	defer fOut.Close()

	n, err := io.Copy(stream, fOut)
	checkError(err)
	fmt.Printf("send %d bytes over LoRa\n", n)

	// Dump 256 bytes received via LoRa to a tempfile
	fIn, err := os.CreateTemp("", "lora_dump_")
	checkError(err)
	defer fIn.Close()

	n, err = io.Copy(fIn, io.LimitReader(stream, 256))
	checkError(err)
	fmt.Printf("wrote %d bytes received over LoRa to %s\n", n, fIn.Name())

	checkError(modem.Close())
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


[godoc]: https://pkg.go.dev/github.com/dtn7/rf95modem-go/rf95
[rf95modem]: https://github.com/gh0st42/rf95modem
[rf95modem-commit]: https://github.com/gh0st42/rf95modem/commit/117878a4b609f9488ad8d5176f98067b9e8baa01
[rf95modem-pr16]: https://github.com/gh0st42/rf95modem/pull/16
