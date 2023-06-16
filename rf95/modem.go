// Package rf95 provides a library to send and receive LoRa PHY packets over rf95modem.
package rf95

import (
	"bufio"
	"context"
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

// ModemMode is the rf95modem's config mode, specified by AT+MODE.
type ModemMode int

const (
	// MediumRange is the default mode for medium range. Bw = 125 kHz, Cr = 4/5, Sf = 128chips/symbol, CRC on.
	MediumRange ModemMode = 0

	// FastShortRange is a fast and short range mode. Bw = 500 kHz, Cr = 4/5, Sf = 128chips/symbol, CRC on.
	FastShortRange ModemMode = 1

	// SlowLongRange is a slow and long range mode. Bw = 31.25 kHz, Cr = 4/8, Sf = 512chips/symbol, CRC on.
	SlowLongRange ModemMode = 2

	// SlowLongRange2 is another slow and long range mode. Bw = 125 kHz, Cr = 4/8, Sf = 4096chips/symbol, CRC on.
	SlowLongRange2 ModemMode = 3

	// SlowLongRange3 is another slow and long range mode. Bw = 125 kHz, Cr = 4/5, Sf = 2048chips/symbol, CRC on.
	SlowLongRange3 ModemMode = 4

	// maxModemMode holds the greatest integer of a known ModemMode used for range checks.
	maxModemMode = int(SlowLongRange3)
)

// RxMessage represents a received message with its fields.
type RxMessage struct {
	Payload []byte
	Rssi    int
	Snr     int
}

// Status describes the rf95modem's status, acquired by AT+INFO.
type Status struct {
	Firmware  string
	Features  []string
	Mode      ModemMode
	Mtu       int
	Frequency float64
	Bfb       int
	RxBad     int
	RxGood    int
	TxGood    int
}

// Modem manages the connection to a rf95modem.
//
// After creation, it's state can be fetched or altered. New handler can be
// registered for data reception and raw data can be send.
type Modem struct {
	devReader io.Reader
	devWriter io.Writer
	devCloser io.Closer

	rxHandlers   []func(RxMessage)
	mtuHandlers  []func(int)
	handlerMutex sync.RWMutex

	atCommandMutex sync.Mutex
	msgQueue       chan string

	ctx       context.Context
	ctxCancel context.CancelFunc
}

// OpenModem creates a new Modem backed by some stream.
//
// Both the io.Reader as well as the io.Writer are necessary. The io.Closer
// might be nil. The Modem finishes when the Context is done.
func OpenModem(r io.Reader, w io.Writer, c io.Closer, ctx context.Context) (modem *Modem, err error) {
	modem = &Modem{
		devReader: r,
		devWriter: w,
		devCloser: c,
		msgQueue:  make(chan string, 128),
	}

	modem.ctx, modem.ctxCancel = context.WithCancel(ctx)

	go modem.worker()

	return
}

// OpenSerial creates a new Modem based on a serial connection to a rf95modem.
//
// The device parameter might be /dev/ttyUSB0, or your operating system's
// equivalent. For Context information, check OpenModem's documentation.
func OpenSerial(device string, ctx context.Context) (modem *Modem, err error) {
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

	return OpenModem(serialPort, serialPort, serialPort, ctx)
}

// parsePacketRx tries to extract the fields of an RX message.
func parsePacketRx(msg string) (rx RxMessage, err error) {
	rxRegexp := regexp.MustCompile(`^\+RX \d+,([0-9A-Fa-f]+),([-0-9]+),([-0-9]+)\r?\n$`)
	findings := rxRegexp.FindStringSubmatch(msg)
	if len(findings) != 4 {
		err = fmt.Errorf("found no matching RX fields")
		return
	}

	if rx.Payload, err = hex.DecodeString(findings[1]); err != nil {
		return
	} else if rx.Rssi, err = strconv.Atoi(findings[2]); err != nil {
		return
	} else if rx.Snr, err = strconv.Atoi(findings[3]); err != nil {
		return
	}

	return
}

// worker reads the input stream and runs within a Goroutine after OpenModem.
//
// Received data will either be distributed to all RX handlers or added to the
// msgQueue when needed for other tasks.
func (modem *Modem) worker() {
	var reader = bufio.NewReader(modem.devReader)

	for {
		select {
		case <-modem.ctx.Done():
			if modem.devCloser != nil {
				_ = modem.devCloser.Close()
			}
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
					modem.handlerMutex.RLock()
					for _, rxHandler := range modem.rxHandlers {
						rxHandler(rxMsg)
					}
					modem.handlerMutex.RUnlock()
				}
			} else {
				modem.msgQueue <- lineMsg
			}
		}
	}
}

// Close down the internal worker and Closer if not nil.
func (modem *Modem) Close() (err error) {
	modem.ctxCancel()

	modem.handlerMutex.Lock()
	modem.rxHandlers = nil
	modem.mtuHandlers = nil
	modem.handlerMutex.Unlock()

	return nil
}

// RegisterHandlers for RxMessages and MTU updates.
//
// Each handler might be nil and thus won't be registered. The returned Context
// will be done if the Modem is finished.
func (modem *Modem) RegisterHandlers(rxHandler func(RxMessage), mtuHandler func(int)) (context.Context, error) {
	modem.handlerMutex.Lock()
	if rxHandler != nil {
		modem.rxHandlers = append(modem.rxHandlers, rxHandler)
	}
	if mtuHandler != nil {
		modem.mtuHandlers = append(modem.mtuHandlers, mtuHandler)
	}
	modem.handlerMutex.Unlock()

	err := modem.refreshMtu()
	if err != nil {
		return nil, err
	}

	return modem.ctx, nil
}

// atCommand executes an AT command and reads lines until stopFn returns false.
//
// The last line where stopFn returns false will also be included in lines.
func (modem *Modem) atCommand(cmd string, stopFn func(string) bool) (lines []string, err error) {
	modem.atCommandMutex.Lock()
	defer modem.atCommandMutex.Unlock()

	_, err = modem.devWriter.Write([]byte(cmd + "\n"))
	if err != nil {
		return
	}

	for {
		select {
		case <-modem.ctx.Done():
			err = io.EOF
			return

		case line := <-modem.msgQueue:
			lines = append(lines, line)
			if !stopFn(line) {
				return
			}
		}
	}
}

// atCommandOnce executes an AT command and reads back one line.
func (modem *Modem) atCommandOnce(cmd string) (string, error) {
	lines, err := modem.atCommand(cmd, func(string) bool { return false })
	if err != nil {
		return "", err
	}

	return lines[0], nil
}

// Transmit the byte array whose length must be shorter than the Mtu.
//
// To transfer a byte array regardless of its length, create a Stream.
func (modem *Modem) Transmit(p []byte) (int, error) {
	respMsg, cmdErr := modem.atCommandOnce(fmt.Sprintf("AT+TX=%s", hex.EncodeToString(p)))
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

// refreshMtu by querying the status and distributing it to all MTU handlers.
func (modem *Modem) refreshMtu() error {
	status, err := modem.FetchStatus()
	if err != nil {
		return err
	}

	modem.handlerMutex.RLock()
	for _, mtuHandler := range modem.mtuHandlers {
		mtuHandler(status.Mtu)
	}
	modem.handlerMutex.RUnlock()

	return nil
}

// Mode sets the ModemMode.
func (modem *Modem) Mode(mode ModemMode) error {
	if int(mode) < 0 || int(mode) > maxModemMode {
		return fmt.Errorf("modem mode %d is not in [0, %d]", mode, maxModemMode)
	}

	respMsg, cmdErr := modem.atCommandOnce(fmt.Sprintf("AT+MODE=%d", mode))
	if cmdErr != nil {
		return cmdErr
	}
	if !strings.HasPrefix(respMsg, "+OK") {
		return fmt.Errorf("changing modem mode returned unexpected response: %s", respMsg)
	}

	return modem.refreshMtu()
}

// Frequency changes the frequency specified in MHz.
func (modem *Modem) Frequency(frequency float64) error {
	respMsg, cmdErr := modem.atCommandOnce(fmt.Sprintf("AT+FREQ=%.2f", frequency))
	if cmdErr != nil {
		return cmdErr
	}
	if !strings.HasPrefix(respMsg, "+FREQ: ") {
		return fmt.Errorf("changing frequency returned unexpected response: %s", respMsg)
	}

	return modem.refreshMtu()
}

// FetchStatus queries the status information from AT+INFO.
func (modem *Modem) FetchStatus() (status Status, err error) {
	defer func() {
		if err != nil {
			status = Status{}
		}
	}()

	respMsgs, cmdErr := modem.atCommand(
		"AT+INFO",
		func(line string) bool { return !strings.HasPrefix(line, "+OK") })
	if cmdErr != nil {
		err = cmdErr
		return
	}

	for _, respMsg := range respMsgs {
		respMsgFilter := regexp.MustCompile(`^(\+STATUS:|\+OK|)\r?\n$`)
		if respMsgFilter.MatchString(respMsg) {
			continue
		}

		splitRegexp := regexp.MustCompile(`^(.+):[ ]+([^\r]+)\r?\n$`)
		fields := splitRegexp.FindStringSubmatch(respMsg)
		if len(fields) != 3 {
			err = fmt.Errorf("non-empty info line does not satisfy regexp: %s", respMsg)
			return
		}

		key, value := fields[1], strings.TrimSpace(fields[2])

		switch key {
		case "firmware":
			status.Firmware = value

		case "features":
			status.Features = strings.Split(value, " ")
			for i := 0; i < len(status.Features); i++ {
				status.Features[i] = strings.TrimSpace(status.Features[i])
			}

		case "modem config":
			cfgRegexp := regexp.MustCompile(`^(\d+) .*`)
			if cfgFields := cfgRegexp.FindStringSubmatch(value); len(cfgFields) != 2 {
				err = fmt.Errorf("failed to extract momdem config from %s", value)
				return
			} else if cfgModeInt, cfgModeIntErr := strconv.Atoi(cfgFields[1]); cfgModeIntErr != nil {
				err = cfgModeIntErr
				return
			} else if cfgModeInt < 0 || cfgModeInt > maxModemMode {
				err = fmt.Errorf("modem config %d is not in [0, %d]", cfgModeInt, maxModemMode)
				return
			} else {
				status.Mode = ModemMode(cfgModeInt)
			}

		case "frequency":
			if freq, freqErr := strconv.ParseFloat(value, 64); freqErr != nil {
				err = freqErr
				return
			} else {
				status.Frequency = freq
			}

		case "max pkt size", "BFB", "rx bad", "rx good", "tx good":
			v, vErr := strconv.Atoi(value)
			if vErr != nil {
				err = vErr
			}

			switch key {
			case "max pkt size":
				status.Mtu = v
			case "BFB":
				status.Bfb = v
			case "rx bad":
				status.RxBad = v
			case "rx good":
				status.RxGood = v
			case "tx good":
				status.TxGood = v
			}

		case "rx listener", "GPS":
			// We don't care about those.

		default:
			err = fmt.Errorf("unknown info key value: %s", key)
			return
		}
	}

	return
}
