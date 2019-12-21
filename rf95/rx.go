package rf95

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
)

// RxMessage represents an incoming RX message with its fields from the rf95modem.
type RxMessage struct {
	Payload []byte
	Rssi    int
	Snr     int
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
	modem.rxQueue <- rx
}

// RegisterRxHandler calls the handler function for each incoming RX message.
func (modem *Modem) RegisterRxHandler(handler func(RxMessage)) {
	modem.rxHandlers = append(modem.rxHandlers, handler)
}
