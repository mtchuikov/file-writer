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

type writeCounter struct {
	wr    io.Writer
	count int64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n, err := wc.wr.Write(p)
	wc.count += int64(n)
	return n, err
}

type FileWriter struct {
	mu sync.Mutex

	mode          os.FileMode
	flags         int
	file          *os.File
	backupPostfix string
	compress      bool
	maxSize       uint
	size          uint

	buf          *bufio.Writer
	wc           *writeCounter
	maxBatchSize int
	batchSize    int

	flushTicker  *time.Ticker
	errorHandler func(error)
	done         chan struct{}
}

func NewFileWriter(file string, opts ...Option) (*FileWriter, error) {
	fw := &FileWriter{
		mode:          defaulFileMode,
		flags:         defaulFileFlags,
		backupPostfix: defaultFileBackupPostfix,
		compress:      defaulFileCompress,
		maxSize:       defaulFileMaxSize,

		maxBatchSize: defaulBufMaxBatchSize,
		flushTicker:  time.NewTicker(defaulBufFlushInterval),
		errorHandler: func(err error) {},
	}

	for _, opt := range opts {
		opt(fw)
	}

	f, size, err := fw.openFile(file, fw.mode)
	if err != nil {
		return nil, err
	}

	fw.mu = sync.Mutex{}
	fw.file = f
	fw.size = size

	fw.wc = &writeCounter{
		wr:    fw.file,
		count: 0,
	}
	fw.buf = bufio.NewWriter(fw.file)
	fw.batchSize = 0
	fw.done = make(chan struct{})

	return fw, nil
}

func (fw *FileWriter) Size() (uint, uint) {
	return fw.size, uint(fw.buf.Buffered())
}

func (fw *FileWriter) BatchSize() int {
	return fw.batchSize
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
	f, size, err := fw.openFile(file, m)
	if err != nil {
		return err
	}

	fw.mode = m
	fw.file = f
	fw.size = size

	fw.setBufWriter(f)

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

func (fw *FileWriter) Flush() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.flushBuf()
}

func (fw *FileWriter) Rotate() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.rotateFile()
}

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
