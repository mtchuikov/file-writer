package filewriter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unsafe"

	"github.com/klauspost/compress/gzip"
)

func (fw *FileWriter) getFileSize(file file) (int64, error) {
	stat, err := file.Stat()
	if err != nil {
		err = errors.Unwrap(err)
		return 0, fmt.Errorf(failedToGetFileStats, err)
	}

	return stat.Size(), nil
}

// openFileFn is a wrapper around os.OpenFile that returns a value
// of type file. This wrapper makes it easier to integrate a
// function for creating mock files during testing.
var openFileFn = func(name string, flag int, mode os.FileMode) (file, error) {
	return os.OpenFile(name, flag, mode)
}

func (fw *FileWriter) openFile(name string, mode os.FileMode) error {
	f, err := openFileFn(name, fw.Flags, mode)
	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToOpenLogFile, err)
	}

	size, err := fw.getFileSize(f)
	if err != nil {
		return err
	}

	fw.File = f
	fw.Size = uint(size)

	return nil
}

// setBufWriter sets the underlying io.Writer for the bufio.Writer
// stored in fw.buf by using unsafe pointer arithmetic to access
// its unexported "wr" field. The field offset is defined by
// bufWriterFieldOffset, which is architecture-dependent. It helps
// avoid having to call Reset method of the bufio.Writer when
// rotating the file.
func (fw *FileWriter) setBufWriter(wr io.Writer) {
	bufPtr := unsafe.Pointer(fw.Buf)
	wrPtr := (*io.Writer)(unsafe.Pointer(uintptr(bufPtr) + bufWriterFieldOffset))
	*wrPtr = wr
}

func (fw *FileWriter) compress(backupName string) error {
	dest, err := openFileFn(backupName, defaulFileFlags, fw.Mode)
	if err != nil {
		return err
	}
	defer dest.Close()

	gr := gzip.NewWriter(dest)
	defer gr.Close()

	// Reset the file pointer to the beginning. Without this,
	// using the Write method would advance the pointer, causing
	// subsequent reads to start from the current position rather
	// than from the beginning.
	fw.File.Seek(0, 0)

	_, err = io.Copy(gr, fw.File)
	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToCompressLogFile, err)
	}

	return nil
}

var (
	removeFileFn = func(name string) error {
		return os.Remove(name)
	}

	// renameFileFn is a wrapper around os.Rename that returns a value
	// renames the file. This wrapper makes it easier to integrate a
	// function for renaming mock files during testing.
	renameFileFn = func(oldpath, newpath string) error {
		return os.Rename(oldpath, newpath)
	}

	// currentTime is a variable that holds the function for obtaining
	// the current time. It is extracted into a variable to facilitate
	// testing, allowing it to be replaced with a mock function.
	currentTime = time.Now
)

// rotate performs log file rotation. It closes the current log
// file, renames it with a timestamp postfix, and opens a new
// one with the original name. It also updates the fw.size field to
// the size of the data currently buffered, without taking into
// account the size of the newly created file, cause it assumed to
// be empty.
func (fw *FileWriter) rotateFile() error {
	name := fw.File.Name()

	err := func() error {
		defer fw.File.Close()

		var err error
		if fw.DeleteOld {
			err = removeFileFn(name)
			if err != nil {
				err = errors.Unwrap(err)
				return fmt.Errorf(failedToRemoveLogFile, err)
			}

		} else {
			postfix := currentTime().Format(fw.RotatePostfix)
			backupName := name + "." + postfix

			if fw.Compress {
				backupName = backupName + ".gz"
				err = fw.compress(backupName)
				if err != nil {
					removeFileFn(backupName)
					return err
				}

				removeFileFn(name)

			} else {
				err := renameFileFn(name, backupName)
				if err != nil {
					err = errors.Unwrap(err)
					return fmt.Errorf(failedToRenameLogFile, err)
				}
			}
		}

		return nil
	}()

	if err != nil {
		return err
	}

	f, err := openFileFn(name, fw.Flags, fw.Mode)
	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToOpenLogFile, err)
	}

	fw.File = f
	fw.Size = 0
	fw.Wc.wr = f
	fw.setBufWriter(fw.Wc)

	return nil
}

func (fw *FileWriter) flushBuf() error {
	err := fw.Buf.Flush()

	fw.Size += fw.Wc.flushedBytes
	fw.Wc.flushedBytes = 0

	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToFlushLogBuffer, err)
	}

	return nil
}
