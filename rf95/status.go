package rf95

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Status describes the rf95modem's status, acquired by AT+INFO.
type Status struct {
	Firmware  string
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
	_, _ = fmt.Fprintf(&sb, "mode=%d,", status.Mode)
	_, _ = fmt.Fprintf(&sb, "mtu=%d,", status.Mtu)
	_, _ = fmt.Fprintf(&sb, "frequency=%.2f,", status.Frequency)
	_, _ = fmt.Fprintf(&sb, "big_funky_ble_frames=%d", status.Bfb)
	_, _ = fmt.Fprintf(&sb, "rx_bad=%d,", status.RxBad)
	_, _ = fmt.Fprintf(&sb, "rx_good=%d,", status.RxGood)
	_, _ = fmt.Fprintf(&sb, "tx_good=%d)", status.TxGood)

	return sb.String()
}

// FetchStatus queries the rf95modem's status information.
func (modem *Modem) FetchStatus() (status Status, err error) {
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

		switch value := fields[2]; fields[1] {
		case "firmware":
			status.Firmware = value

		case "modem config":
			cfgRegexp := regexp.MustCompile(`^(\d+) .*`)
			if cfgFields := cfgRegexp.FindStringSubmatch(value); len(cfgFields) != 2 {
				err = fmt.Errorf("failed to extract momdem config from %s", value)
				return
			} else if cfgModeInt, cfgModeIntErr := strconv.Atoi(cfgFields[1]); cfgModeIntErr != nil {
				err = cfgModeIntErr
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

			switch fields[1] {
			case "max pkt size":
				status.Mtu = v
			case "bfb":
				status.Bfb = v
			case "rx bad":
				status.RxBad = v
			case "rx good":
				status.RxGood = v
			case "tx good":
				status.TxGood = v
			}

		case "rx listener":
			// We don't care about this one.

		default:
			err = fmt.Errorf("unknown info key value: %s", fields[1])
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
