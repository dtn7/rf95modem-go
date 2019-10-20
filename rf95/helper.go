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

// sendCmdMultiline sends an AT command to the rf95modem and reads the amount of requested responding lines.
func (modem *Modem) sendCmdMultiline(cmd string, respLines int) (responses []string, err error) {
	if _, writeErr := modem.serialPort.Write([]byte(cmd)); writeErr != nil {
		err = writeErr
		return
	}

	for i := 0; i < respLines; i++ {
		if resp, respErr := modem.reader.ReadString('\n'); respErr != nil {
			err = respErr
			return
		} else {
			responses = append(responses, resp)
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
