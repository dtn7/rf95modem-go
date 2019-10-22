package main

/*
#define _XOPEN_SOURCE 600

#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>
*/
import "C"
import (
	"io"
	"os"
)

// pty opens and provides a pseudoterminal device.
//
// This should work for all POSIX systems, I hope. The code was kind of copied from
// the "os/signal/internal/pty" package.
func pty() (master *os.File, slave string, err error) {
	fd, fdErr := C.posix_openpt(C.O_RDWR)
	if fdErr != nil {
		err = fdErr
		return
	}

	if _, grantErr := C.grantpt(fd); grantErr != nil {
		C.close(fd)
		err = grantErr
		return
	}

	if _, unlockErr := C.unlockpt(fd); unlockErr != nil {
		C.close(fd)
		err = unlockErr
		return
	}

	master = os.NewFile(uintptr(fd), "pty")
	slave = C.GoString(C.ptsname(fd))
	return
}

// streamCopy copies from the src to the dst in an endless loop.
func streamCopy(dst io.Writer, src io.Reader) {
	for {
		if _, err := io.Copy(dst, src); err != io.EOF {
			return
		}
	}
}
