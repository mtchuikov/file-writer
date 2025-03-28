package filewriter

import (
	"os"
	"time"
)

type Option func(*FileWriter)

func WithFileWriterFileMode(mode int) Option {
	return func(fw *FileWriter) {
		fw.mode = os.FileMode(mode)
	}
}

func WithFileWriterMaxSize(size float64) Option {
	return func(fw *FileWriter) {
		fw.maxSize = uint(size * 1024 * 1024)
	}
}

func WithFileWriterCompress(compress bool) Option {
	return func(fw *FileWriter) {
		fw.compress = compress
	}
}

func WithFileWriterFlushInterval(interval time.Duration) Option {
	return func(fw *FileWriter) {
		if interval == 0 {
			fw.flushTicker = nil
			return
		}

		fw.flushTicker = time.NewTicker(interval)
	}
}

func WithFileWriterMaxBatchSize(size int) Option {
	return func(fw *FileWriter) {
		fw.maxBatchSize = size
	}
}
