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

// ModemMode is the rf95modem's config mode, as specified by AT+MODE.
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

// RxMessage represents an incoming RX message with its fields from the rf95modem.
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

func (status Status) String() string {
	var sb strings.Builder

	_, _ = fmt.Fprint(&sb, "Status(", "firmware=", status.Firmware, ",")
	_, _ = fmt.Fprintf(&sb, "features=%s,", strings.Join(status.Features, ","))
	_, _ = fmt.Fprintf(&sb, "mode=%d,", status.Mode)
	_, _ = fmt.Fprintf(&sb, "mtu=%d,", status.Mtu)
	_, _ = fmt.Fprintf(&sb, "frequency=%.2f,", status.Frequency)
	_, _ = fmt.Fprintf(&sb, "big_funky_ble_frames=%d", status.Bfb)
	_, _ = fmt.Fprintf(&sb, "rx_bad=%d,", status.RxBad)
	_, _ = fmt.Fprintf(&sb, "rx_good=%d,", status.RxGood)
	_, _ = fmt.Fprintf(&sb, "tx_good=%d)", status.TxGood)

	return sb.String()
}

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

// Mode sets the rf95modem's modem config.
func (modem *Modem) Mode(mode ModemMode) error {
	cmd := fmt.Sprintf("AT+MODE=%d\n", mode)
	if respMsg, cmdErr := modem.sendCmd(cmd); cmdErr != nil {
		return cmdErr
	} else if !strings.HasPrefix(respMsg, "+OK") {
		return fmt.Errorf("changing modem mode returned unexpected response: %s", respMsg)
	} else {
		return modem.updateMtu()
	}
}

// Frequency changes the rf95modem's frequency, specified in MHz.
func (modem *Modem) Frequency(frequency float64) error {
	cmd := fmt.Sprintf("AT+FREQ=%.2f\n", frequency)
	if respMsg, cmdErr := modem.sendCmd(cmd); cmdErr != nil {
		return cmdErr
	} else if !strings.HasPrefix(respMsg, "+FREQ: ") {
		return fmt.Errorf("changing frequency returned unexpected response: %s", respMsg)
	} else {
		return modem.updateMtu()
	}
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

// rxHandler is the RX message handler for the io.Reader.
func (modem *Modem) rxHandler(rx RxMessage) {
	_, _ = modem.readBuff.Write(rx.Payload)
}

// RegisterRxHandler calls the handler function for each incoming RX message.
func (modem *Modem) RegisterRxHandler(handler func(RxMessage)) {
	modem.rxHandlers = append(modem.rxHandlers, handler)
}

// sendCmdMultiline sends an AT command to the rf95modem and reads the amount of requested responding lines.
func (modem *Modem) sendCmdMultiline(cmd string, respLines int) (responses []string, err error) {
	modem.cmdLock.Lock()
	defer modem.cmdLock.Unlock()

	select {
	case <-modem.stopSyn:
		err = io.EOF

	default:
		if _, writeErr := modem.devWriter.Write([]byte(cmd)); writeErr != nil {
			err = writeErr
			return
		}

		for i := 0; i < respLines; i++ {
			responses = append(responses, <-modem.msgQueue)
		}
	}

	return
}

// sendCmd sends an AT command to the rf95modem and reads the responding line.
func (modem *Modem) sendCmd(cmd string) (response string, err error) {
	if responses, responsesErr := modem.sendCmdMultiline(cmd, 1); responsesErr != nil {
		err = responsesErr
	} else {
		response = responses[0]
	}

	return
}

// updateMtu fetches the current MTU.
func (modem *Modem) updateMtu() (err error) {
	if status, statusErr := modem.FetchStatus(); statusErr != nil {
		err = statusErr
	} else {
		modem.mtu = status.Mtu
	}

	return
}

// FetchStatus queries the rf95modem's status information.
func (modem *Modem) FetchStatus() (status Status, err error) {
	defer func() {
		if err != nil {
			status = Status{}
		}
	}()

	respMsgs, cmdErr := modem.sendCmdMultiline("AT+INFO\n", 13)
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

		key, value := fields[1], fields[2]

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

// Mtu returns the rf95modem's MTU.
func (modem *Modem) Mtu() (mtu int, err error) {
	if modem.mtu == 0 {
		if mtuErr := modem.updateMtu(); mtuErr != nil {
			err = mtuErr
			return
		}
	}

	mtu = modem.mtu
	return
}
