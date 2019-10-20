package rf95

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/tarm/serial"
)

type Modem struct {
	device     string
	serialPort *serial.Port
	reader     *bufio.Reader
	readLock   sync.WaitGroup
}

func OpenModem(device string) (modem *Modem, err error) {
	serialConf := &serial.Config{
		Name: device,
		Baud: 9600,
	}

	serialPort, serialPortErr := serial.OpenPort(serialConf)
	if serialPortErr != nil {
		err = serialPortErr
		return
	}

	modem = &Modem{
		device:     device,
		serialPort: serialPort,
		reader:     bufio.NewReader(serialPort),
	}

	return
}

func (modem *Modem) Read(p []byte) (int, error) {
	modem.readLock.Wait()

	lineMsg, lineErr := modem.reader.ReadString('\n')
	if lineErr != nil {
		return 0, lineErr
	}

	rxBytes, rxErr := parsePacketRx(lineMsg)
	if rxErr != nil {
		return 0, rxErr
	}

	return copy(p, rxBytes), nil
}

func (modem *Modem) Write(p []byte) (int, error) {
	modem.readLock.Add(1)
	defer modem.readLock.Done()

	msg := fmt.Sprintf("AT+TX=%s\n", hex.EncodeToString(p))
	if _, err := modem.serialPort.Write([]byte(msg)); err != nil {
		return 0, err
	}

	respMsg, respErr := modem.reader.ReadString('\n')
	if respErr != nil {
		return 0, respErr
	}

	respPattern := regexp.MustCompile(`^\+SENT (\d+) bytes\.\r\n$`)
	respMatch := respPattern.FindStringSubmatch(respMsg)
	if len(respMatch) != 2 {
		return 0, fmt.Errorf("unexpected response: %s", respMsg)
	} else if n, nErr := strconv.Atoi(respMatch[1]); nErr != nil {
		return 0, nErr
	} else {
		return n, nil
	}
}

func (modem *Modem) Close() error {
	modem.readLock.Wait()
	return modem.serialPort.Close()
}
