package filewriter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// file is an interface that simplifies testing code that deals
// with files. Instead of using a concrete type like *os.File,
// it's better to substitute stubs or mock objects that don't
// interact with the real filesystem.
type file interface {
	Name() string
	Write(p []byte) (int, error)
	Stat() (os.FileInfo, error)
	Close() error
}

var _ io.WriteCloser = (*FileWriter)(nil)

type FileWriter struct {
	mu sync.Mutex

	mode          os.FileMode
	flags         int
	file          file
	rotatePostfix string // the postfix added to the file name during log rotation
	compress      bool   // indicates whether the log file should be compressed
	maxSize       uint   // the maximum allowed size of the log file (in bytes)
	size          uint   // the current size of the log file + buffer size (in bytes)

	buf          *bufio.Writer
	wc           *writeCounter
	maxBatchSize int // the maximum number of log entries to accumulate before flushing
	batchSize    int // the current number of log entries in the buffer

	flushTicker  *time.Ticker // the time.Ticker that triggers periodic flushes of the buffer
	errorHandler func(error)  // the function to handle errors that occur during flushing
	done         chan struct{}
}

func NewFileWriter(file string, opts ...Option) (*FileWriter, error) {
	fw := &FileWriter{
		mode:          defaulFileMode,
		flags:         defaulFileFlags,
		rotatePostfix: defaultFileRotatePostfix,
		compress:      defaulFileCompress,
		maxSize:       defaulFileMaxSize,

		maxBatchSize: defaulBufMaxBatchSize,
		flushTicker:  time.NewTicker(defaulBufFlushInterval),
		errorHandler: func(err error) {},
	}

	for _, opt := range opts {
		opt(fw)
	}

	err := fw.openFile(file, fw.mode)
	if err != nil {
		return nil, err
	}

	fw.mu = sync.Mutex{}
	fw.wc = &writeCounter{
		wr:    fw.file,
		count: 0,
	}
	fw.buf = bufio.NewWriter(fw.file)
	fw.batchSize = 0
	fw.done = make(chan struct{})

	return fw, nil
}

// Open opens a new log file with the specified name, using the
// flags and permissions set in the FileWriter. It resets the
// internal writer to work with the new file. Before calling
// this function, ensure that the Close method is called to
// properly close the previous log file and avoid resource
// leaks or data corruption
func (fw *FileWriter) Open(file string, mode int) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	m := os.FileMode(mode)
	err := fw.openFile(file, m)
	if err != nil {
		return err
	}

	fw.setBufWriter(fw.file)

	return nil
}

// Write writes the provided data to the log file while ensuring
// that the total size of the file, the buffered data, and the
// new data does not exceed the maximum allowed size. If the new
// data would cause the size to surpass this limit, the log file
// is rotated and any buffered data is flushed before proceeding.
// After writing, if the number of batched entries reaches the
// predefined threshold, the buffer is flushed
func (fw *FileWriter) Write(p []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	pSize := uint(len(p))
	size := fw.size + pSize

	var err error
	if size >= fw.maxSize {
		fw.batchSize = 0
		err = fw.rotateFile()
		if err != nil {
			return 0, err
		}

		err = fw.flushBuf()
		if err != nil {
			return 0, err
		}
	}

	n, err := fw.buf.Write(p)
	fw.size += uint(n)
	if err != nil {
		err = errors.Unwrap(err)
		return n, fmt.Errorf(failedToWriteLogFile, err)
	}

	fw.batchSize++
	if fw.batchSize >= fw.maxBatchSize {
		fw.batchSize = 0
		// flush the buffer without rotating, because after the
		// postWriteSize calculation we assume that there will be enough
		// space after rotation, but in some cases (e.g., when an
		// excessively large number of bytes is passed), this assumption
		// might not hold; it is the user's responsibility to ensure that
		// the input size remains within acceptable limits
		err = fw.flushBuf()
	}

	return n, err
}

// Close terminates the FileWriter by stopping the periodic flush
// ticker, closing the done channel, and then ensuring that any
// buffered log data is properly handled before the file is closed.
// It calculates the total size as the sum of the current file size
// and the number of bytes buffered. If this total exceeds the
// maximum allowed size, the log file is rotated. If no error ccurs
// during rotation, the remaining buffered data is flushed to the
// file.
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer func() {
		fw.file.Close()
		fw.mu.Unlock()
	}()

	fw.flushTicker.Stop()
	close(fw.done)

	bufSize := uint(fw.buf.Buffered())
	size := fw.size + bufSize

	var err error
	if size > fw.maxSize {
		err = fw.rotateFile()
	}

	if err == nil {
		fw.flushBuf()
	}

	return err
}
