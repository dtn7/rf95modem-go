package rf95

import (
	"bufio"
	"sync"

	"github.com/tarm/serial"
)

type Modem struct {
	device     string
	serialPort *serial.Port
	reader     *bufio.Reader
	lock       sync.WaitGroup
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
	modem.lock.Wait()

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

func (modem *Modem) Close() error {
	modem.lock.Wait()
	return modem.serialPort.Close()
}
