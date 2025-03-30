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
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Stat() (os.FileInfo, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

var _ io.WriteCloser = (*FileWriter)(nil)

type FileWriter struct {
	mu sync.Mutex

	Mode          os.FileMode
	Flags         int
	File          file
	DeleteOld     bool   // indicates whether the old log file should be removed after rotation
	RotatePostfix string // the postfix added to the file name during log rotation
	Compress      bool   // indicates whether the log file should be compressed
	MaxSize       uint   // the maximum allowed size of the log file (in bytes)
	Size          uint   // the current size of the log file + buffer size (in bytes)

	Buf          *bufio.Writer
	Wc           *writeCounter
	MaxBatchSize int // the maximum number of log entries to accumulate before flushing
	BatchSize    int // the current number of log entries in the buffer

	// the time.Ticker that triggers periodic flushes of the buffer
	FlushTicker *time.Ticker
	// the function to handle errors that occur during flushing
	ErrorHandler func(fw *FileWriter, err error)
	Done         chan struct{}
}

func (fw *FileWriter) runTicker() {
	if fw.FlushTicker == nil {
		return
	}

	go func() {
		for {
			select {
			case <-fw.Done:
				return
			case <-fw.FlushTicker.C:
				fw.mu.Lock()

				err := func() error {
					bufSize := uint(fw.Buf.Buffered())
					afterWriteSize := fw.Size + bufSize

					var err error
					if afterWriteSize >= fw.MaxSize {
						err = fw.rotateFile()
					}

					if err == nil {
						fw.BatchSize = 0
						err = fw.flushBuf()
					}

					return err
				}()

				if err != nil {
					fw.ErrorHandler(fw, err)
				}

				fw.mu.Unlock()
			}
		}
	}()
}

func NewFileWriter(file string, opts ...Option) (*FileWriter, error) {
	fw := &FileWriter{
		Mode:          defaulFileMode,
		Flags:         defaulFileFlags,
		DeleteOld:     defaultFileDeleteOld,
		RotatePostfix: defaultFileRotatePostfix,
		Compress:      defaulFileCompress,
		MaxSize:       defaulFileMaxSize,

		MaxBatchSize: defaulBufMaxBatchSize,
		FlushTicker:  time.NewTicker(defaulBufFlushInterval),
		ErrorHandler: func(fw *FileWriter, err error) {},
	}

	for _, opt := range opts {
		opt(fw)
	}

	err := fw.openFile(file, fw.Mode)
	if err != nil {
		return nil, err
	}

	fw.mu = sync.Mutex{}
	fw.Wc = &writeCounter{wr: fw.File}
	fw.Buf = bufio.NewWriter(fw.Wc)

	fw.BatchSize = 0
	fw.Done = make(chan struct{})

	fw.runTicker()

	return fw, nil
}

// Open opens a new log file with the specified name, using the
// flags and permissions set in the FileWriter. It resets the
// internal writer to work with the new file. Before calling
// this function, ensure that the Close method is called to
// properly close the previous log file and avoid resource
// leaks or data corruption.
func (fw *FileWriter) Open(file string, mode int) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	m := os.FileMode(mode)
	err := fw.openFile(file, m)
	if err != nil {
		return err
	}

	fw.Mode = m
	fw.setBufWriter(fw.File)
	fw.Done = make(chan struct{})

	fw.runTicker()

	return nil
}

// Write writes the provided data to the log file while ensuring
// that the total size of the file, the buffered data, and the
// new data does not exceed the maximum allowed size. If the new
// data would cause the size to surpass this limit, the log file
// is rotated and any buffered data is flushed before proceeding.
// After writing, if the number of batched entries reaches the
// predefined threshold, the buffer is flushed.
func (fw *FileWriter) Write(p []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	pSize := uint(len(p))
	bufSize := uint(fw.Buf.Buffered())
	afterWriteSize := fw.Size + pSize + bufSize

	var err error
	if afterWriteSize >= fw.MaxSize {
		err = fw.rotateFile()
		if err != nil {
			return 0, err
		}

		err = fw.flushBuf()
		if err != nil {
			return 0, err
		}

		fw.BatchSize = 0
	}

	n, err := fw.Buf.Write(p)
	if err != nil {
		err = errors.Unwrap(err)
		return n, fmt.Errorf(failedToWriteLogFile, err)
	}

	fw.BatchSize++
	if fw.BatchSize >= fw.MaxBatchSize {
		fw.BatchSize = 0
		// Flush the buffer without rotating, because after the
		// postWriteSize calculation we assume that there will be enough
		// space after rotation, but in some cases (e.g., when an
		// excessively large number of bytes is passed), this assumption
		// might not hold; it is the user's responsibility to ensure that
		// the input size remains within acceptable limits.
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
		fw.File.Close()
		fw.mu.Unlock()
	}()

	fw.FlushTicker.Stop()
	close(fw.Done)

	bufSize := uint(fw.Buf.Buffered())
	afterWriteSize := fw.Size + bufSize

	var err error
	if afterWriteSize >= fw.MaxSize {
		err = fw.rotateFile()
		if err != nil {
			return err
		}
	}

	return fw.flushBuf()
}
