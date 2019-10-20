package rf95

import (
	"bufio"
	"bytes"
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
	readBuff   *bytes.Buffer
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
		readBuff:   new(bytes.Buffer),
	}

	return
}

// Read the next received message in the given byte array.
//
// If the byte array's length is shorter than that of the message, the message's data is cached and read on
// the next call. Should the cache be empty, this method blocks until data is received.
func (modem *Modem) Read(p []byte) (int, error) {
	if modem.readBuff.Len() > 0 {
		return modem.readBuff.Read(p)
	}

	modem.readLock.Wait()

	lineMsg, lineErr := modem.reader.ReadString('\n')
	if lineErr != nil {
		return 0, lineErr
	}

	rxBytes, rxErr := parsePacketRx(lineMsg)
	if rxErr != nil {
		return 0, rxErr
	}

	_, _ = modem.readBuff.Write(rxBytes)
	return modem.readBuff.Read(p)
}

func (modem *Modem) Write(p []byte) (int, error) {
	modem.readLock.Add(1)
	defer modem.readLock.Done()

	cmd := fmt.Sprintf("AT+TX=%s\n", hex.EncodeToString(p))
	respMsg, cmdErr := modem.sendCmd(cmd)
	if cmdErr != nil {
		return 0, cmdErr
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
