package filewriter

import "runtime"

// bufWriterFieldOffset represents the byte offset of the
// unexported "wr" field within the Writer struct from the standard
// library's bufio package.
var bufWriterFieldOffset uintptr

func init() {
	arch := runtime.GOARCH
	if arch == "amd64" {
		bufWriterFieldOffset = 48
		return
	}

	bufWriterFieldOffset = 24
}
