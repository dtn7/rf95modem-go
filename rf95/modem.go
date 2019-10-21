// Package rf95 provides a software interface for a rf95modem. This allows packets to be received and sent
// via the known Reader / Writer interfaces.
package rf95

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// Modem is a software library around the connection to a rf95modem. Thus, a Modem might receive and
// transmit data via LoRa. Furthermore, both Reader and Writer are implemented.
type Modem struct {
	device     string
	serialPort *serial.Port
	readBuff   *bytes.Buffer
	rxQueue    chan string
	msgQueue   chan string
	stopSyn    chan struct{}
	stopAck    chan struct{}
	mtu        int
}

// OpenModem tries to create a new Modem for a given device. This device must be a serial connection with
// a rf95modem being active on the other side. Possible parameters might be /dev/ttyUSB0, or your operating
// system's equivalent.
func OpenModem(device string) (modem *Modem, err error) {
	serialConf := &serial.Config{
		Name:        device,
		Baud:        9600,
		ReadTimeout: time.Second,
	}

	serialPort, serialPortErr := serial.OpenPort(serialConf)
	if serialPortErr != nil {
		err = serialPortErr
		return
	}

	modem = &Modem{
		device:     device,
		serialPort: serialPort,
		readBuff:   new(bytes.Buffer),
		rxQueue:    make(chan string, 32),
		msgQueue:   make(chan string, 32),
		stopSyn:    make(chan struct{}),
		stopAck:    make(chan struct{}),
	}

	go modem.handleRead()
	return
}

// handleRead dispatches the inbounding data to the rxQueue for received messages and msgQueue for everything else.
func (modem *Modem) handleRead() {
	var reader = bufio.NewReader(modem.serialPort)
	for {
		select {
		case <-modem.stopSyn:
			return

		default:
			lineMsg, lineErr := reader.ReadString('\n')
			if lineErr == io.EOF {
				continue
			} else if lineErr != nil {
				return
			}

			if strings.HasPrefix(lineMsg, "+RX") {
				modem.rxQueue <- lineMsg
			} else {
				modem.msgQueue <- lineMsg
			}
		}
	}
}

// Read the next received message in the given byte array.
//
// If the byte array's length is shorter than that of the message, the message's data is cached and read on
// the next call. Should the cache be empty, this method blocks until data is received.
func (modem *Modem) Read(p []byte) (int, error) {
	if modem.readBuff.Len() > 0 {
		return modem.readBuff.Read(p)
	}

	lineMsg := <-modem.rxQueue
	rxBytes, rxErr := parsePacketRx(lineMsg)
	if rxErr != nil {
		return 0, rxErr
	}

	_, _ = modem.readBuff.Write(rxBytes)
	return modem.readBuff.Read(p)
}

// Transmit the byte array whose length must be shorter than the Mtu. To transfer a byte array regardless
// of its length, use Write.
func (modem *Modem) Transmit(p []byte) (int, error) {
	cmd := fmt.Sprintf("AT+TX=%s\n", hex.EncodeToString(p))
	respMsg, cmdErr := modem.sendCmd(cmd)
	if cmdErr != nil {
		return 0, cmdErr
	}

	respPattern := regexp.MustCompile(`^\+SENT (\d+) bytes\.\r\n$`)
	respMatch := respPattern.FindStringSubmatch(respMsg)
	if len(respMatch) != 2 {
		return 0, fmt.Errorf("unexpected response: %s", respMsg)
	} else if n, nErr := strconv.Atoi(respMatch[1]); nErr != nil {
		return 0, nErr
	} else {
		return n, nil
	}
}

// Write the byte array to the rf95modem. If its length exceeds the Mtu, multiple packets will be send.
func (modem *Modem) Write(p []byte) (n int, err error) {
	if _, mtuErr := modem.Mtu(); mtuErr != nil {
		err = mtuErr
		return
	}

	for i := 0; i < len(p); i += modem.mtu {
		bound := i + modem.mtu
		if bound > len(p) {
			bound = len(p)
		}

		tx, txErr := modem.Transmit(p[i:bound])
		n += tx
		if txErr != nil {
			err = txErr
			return
		}
	}

	return
}

// Close the underlying serial connection.
func (modem *Modem) Close() error {
	close(modem.stopSyn)
	<-modem.stopAck

	return modem.serialPort.Close()
}
