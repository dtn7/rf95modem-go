package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/dtn7/rf95modem-go/rf95"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage:   %s DEVICE\n", os.Args[0])
		fmt.Printf("Example: %s /dev/ttyUSB0\n\n", os.Args[0])
		os.Exit(1)
	}

	sigintCtx, sigintCtxCancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer sigintCtxCancel()

	modem, modemErr := rf95.OpenSerial(os.Args[1], sigintCtx)
	if modemErr != nil {
		panic(modemErr)
	}

	if status, statusErr := modem.FetchStatus(); statusErr != nil {
		panic(statusErr)
	} else {
		fmt.Printf("Starting modem with %#v\n", status)
	}

	stream, streamErr := rf95.NewStream(modem)
	if streamErr != nil {
		panic(streamErr)
	}

	ptyMaster, ptySlave, ptyErr := pty()
	if ptyErr != nil {
		panic(ptyErr)
	}

	fmt.Printf("Opening pty device %s\n", ptySlave)

	go streamCopy(stream, ptyMaster)
	go streamCopy(ptyMaster, stream)

	<-sigintCtx.Done()

	if err := modem.Close(); err != nil {
		fmt.Printf("Closing errored: %v\n", err)
	}
}
