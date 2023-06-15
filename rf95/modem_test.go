package rf95

import (
	"reflect"
	"testing"
)

func TestParseRxMessage(t *testing.T) {
	tests := []struct {
		msg    string
		errors bool
		rx     RxMessage
	}{
		{"+RX 3,414141,-15,8\n", false, RxMessage{[]byte{0x41, 0x41, 0x41}, -15, 8}},
		{"+RX 3,ACAB,23,42\n", false, RxMessage{[]byte{0xAC, 0xAB}, 23, 42}},
		{"+RX 3,XYZ,23,42\n", true, RxMessage{}},
		{"+RX 3,1234,F3,42\n", true, RxMessage{}},
		{"+RX 3,1234,23,F2\n", true, RxMessage{}},
		{"+RX ,1234,23,32\n", true, RxMessage{}},
	}

	for _, test := range tests {
		if rx, err := parsePacketRx(test.msg); (err == nil) == test.errors {
			t.Fatalf("RX message \"%s\" returned error %v, expected %t", test.msg, err, test.errors)
		} else if !reflect.DeepEqual(rx, test.rx) {
			t.Fatalf("RX message \"%s\" returned %v, expected %v", test.msg, rx, test.rx)
		}
	}
}
