package filewriter

import (
	"os"
	"time"
)

const (
	// The owner has read, write, and execute permissions, while the
	// group and others have read and execute permissions only.
	defaulFileMode = 0755

	// os.O_CREATE creates the file if it doesn't exist, os.O_WRONLY
	// opens the file for write-only access, and os.O_APPEND ensures
	// that data is always written at the end of the file.
	defaulFileFlags = os.O_CREATE | os.O_RDWR | os.O_APPEND

	defaultFileDeleteOld = false

	// Defines the timestamp format that is appended to the file name
	// after rotation. For example, if the original file name was
	// "test.log", after rotation with the default postfix, it will be
	// renamed to "test.log.2006-01-02T15:04:05Z07:00".
	defaultFileRotatePostfix = time.RFC3339

	// Indicates whether log files should be compressed using gzip,
	// when set to true, logs will be compressed before being saved to
	// the file.
	defaulFileCompress = true

	// The maximum size of the log file in bytes, by the default it
	// equals to 4_194_304 B or 4 MB.
	defaulFileMaxSize = 4 * 1024 * 1024

	// The maximum number of log entries that can be buffered before
	// the logs are flushed.
	defaulBufMaxBatchSize = 64

	// The interval at which the log buffer is flushed to disk, helps
	// to ensure that logs are written periodically even if the batch
	// size is not reached.
	defaulBufFlushInterval = 10 * time.Second
)

const (
	wFailedToOpenLogFile     = "failed to open log file: %w"
	wFailedToRenameLogFile   = "failed to rename log file: %w"
	wFailedToGetFileStats    = "failed to get file stats: %w"
	wFailedToWriteLogFile    = "failed to write log file: %w"
	wFailedToCompressLogFile = "failed to compress log file: %w"
	wFailedToRemoveLogFile   = "failed to remove log file: %w"
	wFailedToFlushLogBuffer  = "failed to flush log buffer: %w"
)
