package filewriter

import (
	"os"
	"time"
)

const (
	// the owner has read, write, and execute permissions, while the
	// group and others have read and execute permissions only
	defaulFileMode = 0755

	// os.O_CREATE creates the file if it doesn't exist, os.O_WRONLY
	// opens the file for write-only access, and os.O_APPEND ensures
	// that data is always written at the end of the file
	defaulFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND

	defaultFileBackupPostfix = time.RFC3339

	// indicates whether log files should be compressed using gzip,
	// when set to true, logs will be compressed before being saved to
	// the file
	defaulFileCompress = true

	// the maximum size of the log file in bytes, by the default it
	// equals to 4 MB
	defaulFileMaxSize = 4 * 1024 * 1024

	// the maximum number of log entries that can be buffered before
	// the logs are flushed
	defaulBufMaxBatchSize = 32

	// the interval at which the log buffer is flushed to disk, helps
	// to ensure that logs are written periodically even if the batch
	// size is not reached
	defaulBufFlushInterval = 10 * time.Second
)

const (
	failedToOpenLogFile   = "failed to open log file: %v"
	failedToRenameLogFile = "failed to rename log file: %v"
	failedToGetFileStats  = "failed to get file stats: %v"
	failedToWriteLogFile  = "failed to write log file: %v"
	failedToFlushLogBuf   = "failed to flush log buffer: %v"
	failedToRotateLogFile = "failed to rotate log file: %v"
)
