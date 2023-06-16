package rf95

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Stream allows using io.Reader and io.Writer around a Modem.
//
// The Stream type automatically handles fragmentations to fit data chunks in
// the Modem's current MTU. It also registers itself as a handlers. When the
// Modem is closed, both Read and Write should report errors.
type Stream struct {
	modem *Modem

	ctx context.Context

	rxBuff      bytes.Buffer
	rxBuffMutex sync.Mutex

	// mtu is protected through sync/atomic calls.
	mtu int32
}

// NewStream backed by the given Modem.
//
// This function registers itself with its handler functions at the Modem.
func NewStream(modem *Modem) (*Stream, error) {
	s := &Stream{modem: modem}

	ctx, err := modem.RegisterHandlers(s.handleRx, s.handleMtu)
	if err != nil {
		return nil, err
	}
	s.ctx = ctx

	return s, nil
}

// handleRx is the rxHandler being passed to the Modem.
func (stream *Stream) handleRx(rx RxMessage) {
	stream.rxBuffMutex.Lock()
	defer stream.rxBuffMutex.Unlock()

	_, _ = stream.rxBuff.Write(rx.Payload)
}

// handleMtu is the mtuHandler passed to the Modem.
func (stream *Stream) handleMtu(mtu int) {
	atomic.StoreInt32(&stream.mtu, int32(mtu))
}

// Read the next received message from the rf95modem in the byte array.
//
// If the byte array's length is shorter than that of the message, the data is
// cached and read on the next call. Should the cache be empty, this method
// blocks until data is received.
func (stream *Stream) Read(p []byte) (int, error) {
	for {
		select {
		case <-stream.ctx.Done():
			return 0, io.EOF

		default:
			stream.rxBuffMutex.Lock()
			if stream.rxBuff.Len() > 0 {
				defer stream.rxBuffMutex.Unlock()
				return stream.rxBuff.Read(p)
			}
			stream.rxBuffMutex.Unlock()

			// TODO: find a more elegant solution
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// Write the byte array to the rf95modem.
//
// If its length exceeds the MTU, multiple packets will be send.
func (stream *Stream) Write(p []byte) (n int, err error) {
	for pos := 0; pos < len(p); {
		mtu := int(atomic.LoadInt32(&stream.mtu))

		bound := pos + mtu
		if bound > len(p) {
			bound = len(p)
		}

		tx, txErr := stream.modem.Transmit(p[pos:bound])
		n += tx
		if txErr != nil {
			err = txErr
			return
		}

		pos += mtu
	}

	return
}
