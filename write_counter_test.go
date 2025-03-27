package filewriter

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteCounter(t *testing.T) {
	var writer bytes.Buffer
	wc := &writeCounter{wr: &writer}

	buf := bufio.NewWriter(wc)

	payload := []byte("Hello, world!\n")
	payloadSize := uint(len(payload))

	buf.Write(payload)
	buf.Flush()

	require.Equal(t,
		payloadSize, wc.flushedBytes,
		"expected flushed bytes to be equal to '%v', got '%v'",
		payloadSize, wc.flushedBytes,
	)
}
