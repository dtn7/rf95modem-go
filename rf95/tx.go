package rf95

import (
	"io"
)

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
