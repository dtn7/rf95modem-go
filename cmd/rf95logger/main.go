package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/dtn7/rf95modem-go/rf95"
)

// waitSigint blocks the current thread until a SIGINT appears.
func waitSigint() {
	signalSyn := make(chan os.Signal)
	signalAck := make(chan struct{})

	signal.Notify(signalSyn, os.Interrupt)

	go func() {
		<-signalSyn
		close(signalAck)
	}()

	<-signalAck
}

// handler prints the received message with its RSSI and SNR as a CSV on the stdout.
func handler(rx rf95.RxMessage) {
	fmt.Printf("%d,%x,%d,%d\n", time.Now().UnixNano(), rx.Payload, rx.Rssi, rx.Snr)
}

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("Usage:   %s DEVICE FREQ MODE-NO\n", os.Args[0])
		fmt.Printf("Example: %s /dev/ttyUSB0 868.5 0\n\n", os.Args[0])
		os.Exit(1)
	}

	modem, modemErr := rf95.OpenSerial(os.Args[1])
	if modemErr != nil {
		panic(modemErr)
	}

	if freq, freqErr := strconv.ParseFloat(os.Args[2], 64); freqErr != nil {
		panic(freqErr)
	} else if freqErr = modem.Frequency(freq); freqErr != nil {
		panic(freqErr)
	}

	if modeNo, modeNoErr := strconv.Atoi(os.Args[3]); modeNoErr != nil {
		panic(modeNoErr)
	} else if modeNoErr = modem.Mode(rf95.ModemMode(modeNo)); modeNoErr != nil {
		panic(modeNoErr)
	}

	fmt.Println("unix_nanosec,payload,rssi,snr")
	modem.RegisterRxHandler(handler)

	waitSigint()

	if closeErr := modem.Close(); closeErr != nil {
		panic(closeErr)
	}
}
