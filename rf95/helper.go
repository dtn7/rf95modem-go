package rf95

import (
	"encoding/hex"
	"fmt"
	"regexp"
)

// parsePacketRx tries to extract the received data from a RX message.
func parsePacketRx(msg string) (data []byte, err error) {
	rxRegexp := regexp.MustCompile(`^\+RX \d+,([0-9A-Fa-f]+),[-0-9]+,\d+\r\n$`)
	findings := rxRegexp.FindStringSubmatch(msg)
	if len(findings) != 2 {
		err = fmt.Errorf("found no matching RX")
		return
	}

	return hex.DecodeString(findings[1])
}

// sendCmd sends an AT command to the rf95modem and reads the responding line.
func (modem *Modem) sendCmd(cmd string) (response string, err error) {
	if _, writeErr := modem.serialPort.Write([]byte(cmd)); writeErr != nil {
		err = writeErr
		return
	}

	response, err = modem.reader.ReadString('\n')
	return
}
