package util

import (
	"io"
)

// A reader that waits for any unavailable bytes to become available for
// reading.  This means that it will always wait until it can fill the entire
// `[]byte` when `Read()` is called, as long as no error occurs.
type WaitReader struct {
	r io.Reader
}

func NewWaitReader(r io.Reader) WaitReader {
	return WaitReader{
		r: r,
	}
}

func (r WaitReader) Read(p []byte) (int, error) {
	var n int // Total number of bytes read.
	for {
		// Reattempt to read the unread bytes until we can fill `p` completely
		// (or an error occurs).
		nNew, err := r.r.Read(p[n:len(p)])
		n += nNew
		if err != nil {
			return n, err
		}
		// Break when we have read enough bytes.
		if n == len(p) {
			break
		}
	}
	return n, nil
}
