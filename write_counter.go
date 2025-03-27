package filewriter

import "io"

// writeCounter wraps an io.Writer and counts the total number of
// bytes written. It also allows to track the size of the write
// operations performed when Flush is called on a buffered writer
type writeCounter struct {
	wr           io.Writer
	flushedBytes uint
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n, err := wc.wr.Write(p)
	wc.flushedBytes += uint(n)
	return n, err
}
