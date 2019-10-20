package rf95

import (
	"encoding/hex"
	"fmt"
	"regexp"
)

func parsePacketRx(msg string) (data []byte, err error) {
	rxRegexp := regexp.MustCompile(`^\+RX \d+,([0-9A-Fa-f]+),[-0-9]+,\d+\r\n$`)
	findings := rxRegexp.FindStringSubmatch(msg)
	if len(findings) != 2 {
		err = fmt.Errorf("found no matching RX")
		return
	}

	return hex.DecodeString(findings[1])
}
