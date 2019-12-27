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
	"sync"
	"time"

	"github.com/tarm/serial"
)

// Modem is a software library around the connection to a rf95modem. Thus, a Modem might receive and
// transmit data via LoRa. Furthermore, both Reader and Writer are implemented.
type Modem struct {
	devReader  io.Reader
	devWriter  io.Writer
	devCloser  io.Closer
	readBuff   *bytes.Buffer
	cmdLock    sync.Mutex
	rxHandlers []func(RxMessage)
	msgQueue   chan string
	stopSyn    chan struct{}
	stopAck    chan struct{}
	mtu        int
}

// OpenModem creates a new Modem, connected to a Reader and a Writer. The Closer might be nil.
func OpenModem(r io.Reader, w io.Writer, c io.Closer) (modem *Modem, err error) {
	modem = &Modem{
		devReader: r,
		devWriter: w,
		devCloser: c,
		readBuff:  new(bytes.Buffer),
		msgQueue:  make(chan string, 128),
		stopSyn:   make(chan struct{}),
		stopAck:   make(chan struct{}),
	}

	modem.RegisterRxHandler(modem.rxHandler)
	go modem.handleRead()

	return
}

// OpenSerial creates a new Modem from a serial connection to a rf95modem. The device parameter might be
// /dev/ttyUSB0, or your operating system's equivalent.
func OpenSerial(device string) (modem *Modem, err error) {
	serialConf := &serial.Config{
		Name:        device,
		Baud:        115200,
		ReadTimeout: time.Second,
	}

	serialPort, serialPortErr := serial.OpenPort(serialConf)
	if serialPortErr != nil {
		err = serialPortErr
		return
	}

	return OpenModem(serialPort, serialPort, serialPort)
}

// handleRead dispatches the inbounding data to the rxQueue for received messages and msgQueue for everything else.
func (modem *Modem) handleRead() {
	var reader = bufio.NewReader(modem.devReader)
	for {
		select {
		case <-modem.stopSyn:
			close(modem.stopAck)
			return

		default:
			lineMsg, lineErr := reader.ReadString('\n')
			if lineErr == io.EOF {
				continue
			} else if lineErr != nil {
				return
			}

			if strings.HasPrefix(lineMsg, "+RX") {
				if rxMsg, rxErr := parsePacketRx(lineMsg); rxErr == nil {
					for _, h := range modem.rxHandlers {
						h(rxMsg)
					}
				}
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

	for {
		select {
		case <-modem.stopSyn:
			return 0, io.EOF

		default:
			if modem.readBuff.Len() > 0 {
				return modem.readBuff.Read(p)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// Transmit the byte array whose length must be shorter than the Mtu. To transfer a byte array regardless
// of its length, use Write.
func (modem *Modem) Transmit(p []byte) (int, error) {
	cmd := fmt.Sprintf("AT+TX=%s\n", hex.EncodeToString(p))
	respMsg, cmdErr := modem.sendCmd(cmd)
	if cmdErr != nil {
		return 0, cmdErr
	}

	respPattern := regexp.MustCompile(`^\+SENT (\d+) bytes\.\r?\n$`)
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
func (modem *Modem) Close() (err error) {
	close(modem.stopSyn)
	<-modem.stopAck

	if modem.devCloser != nil {
		err = modem.devCloser.Close()
	}

	modem.rxHandlers = nil

	return
}
