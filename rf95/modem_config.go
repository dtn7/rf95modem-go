package rf95

import (
	"fmt"
	"strings"
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
)

// Mode sets the rf95modem's modem config.
func (modem *Modem) Mode(mode ModemMode) error {
	modem.readLock.Add(1)
	defer modem.readLock.Done()

	cmd := fmt.Sprintf("AT+MODE=%d\n", mode)
	if respMsg, cmdErr := modem.sendCmd(cmd); cmdErr != nil {
		return cmdErr
	} else if !strings.HasPrefix(respMsg, "+ Ok.") {
		return fmt.Errorf("changing modem mode returned unexpected response: %s", respMsg)
	} else {
		return nil
	}
}

// Frequency changes the rf95modem's frequency, specified in MHz.
func (modem *Modem) Frequency(frequency float64) error {
	modem.readLock.Add(1)
	defer modem.readLock.Done()

	cmd := fmt.Sprintf("AT+FREQ=%.2f\n", frequency)
	if respMsg, cmdErr := modem.sendCmd(cmd); cmdErr != nil {
		return cmdErr
	} else if !strings.HasPrefix(respMsg, "Set Freq to: ") {
		return fmt.Errorf("changing frequency returned unexpected response: %s", respMsg)
	} else {
		return nil
	}
}
