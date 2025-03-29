package filewriter

import (
	"os"
	"time"
)

type Option func(*FileWriter)

func WithFileWriterFileMode(mode int) Option {
	return func(fw *FileWriter) {
		fw.Mode = os.FileMode(mode)
	}
}

func WithFileDeleteOld(delete bool) Option {
	return func(fw *FileWriter) {
		fw.DeleteOld = delete
	}
}

func WithFileRotatePostfix(postfix string) Option {
	return func(fw *FileWriter) {
		fw.RotatePostfix = postfix
	}
}

func WithFileCompress(compress bool) Option {
	return func(fw *FileWriter) {
		fw.Compress = compress
	}
}

func WithFileMaxSize(size float64) Option {
	return func(fw *FileWriter) {
		fw.MaxSize = uint(size * 1024 * 1024)
	}
}

func WithLogMaxBatchSize(size int) Option {
	return func(fw *FileWriter) {
		fw.MaxBatchSize = size
	}
}

func WithLogFlushInterval(interval time.Duration) Option {
	return func(fw *FileWriter) {
		if interval == 0 {
			fw.FlushTicker = nil
			return
		}

		fw.FlushTicker = time.NewTicker(interval)
	}
}

func WithErrorHandler(h func(fw *FileWriter, err error)) Option {
	return func(fw *FileWriter) {
		fw.ErrorHandler = h
	}
}
