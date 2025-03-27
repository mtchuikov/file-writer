package filewriter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unsafe"
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
// function for creating mock files during testing
var openFileFn = func(name string, flag int, mode os.FileMode) (file, error) {
	return os.OpenFile(name, flag, mode)
}

func (fw *FileWriter) openFile(name string, mode os.FileMode) error {
	f, err := openFileFn(name, fw.flags, mode)
	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToOpenLogFile, err)
	}

	size, err := fw.getFileSize(f)
	if err != nil {
		return err
	}

	fw.file = f
	fw.size = uint(size)

	return nil
}

// setBufWriter sets the underlying io.Writer for the bufio.Writer
// stored in fw.buf by using unsafe pointer arithmetic to access
// its unexported "wr" field. The field offset is defined by
// bufWriterFieldOffset, which is architecture-dependent. It helps
// avoid having to call Reset method of the bufio.Writer when
// rotating the file
func (fw *FileWriter) setBufWriter(wr io.Writer) {
	bufPtr := unsafe.Pointer(fw.buf)
	wrPtr := (*io.Writer)(unsafe.Pointer(uintptr(bufPtr) + bufWriterFieldOffset))
	*wrPtr = wr
}

// renameFileFn is a wrapper around os.Rename that returns a value
// renames the file. This wrapper makes it easier to integrate a
// function for renaming mock files during testing
var renameFileFn = func(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// rotate performs log file rotation. It closes the current log
// file, renames it with a timestamp postfix, and opens a new
// one with the original name. It also updates the fw.size field to
// the size of the data currently buffered, without taking into
// account the size of the newly created file, cause it assumed to
// be empty
func (fw *FileWriter) rotateFile() error {
	name := fw.file.Name()
	fw.file.Close()

	postfix := time.Now().Format(fw.rotatePostfix)
	backupName := name + "." + postfix

	err := renameFileFn(name, backupName)
	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToRenameLogFile, err)
	}

	f, err := openFileFn(name, fw.flags, fw.mode)
	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToOpenLogFile, err)
	}

	fw.file = f
	fw.size = uint(fw.buf.Buffered())

	fw.wc.wr = f
	fw.setBufWriter(fw.wc)

	return nil
}

func (fw *FileWriter) flushBuf() error {
	bufSize := fw.buf.Buffered()
	err := fw.buf.Flush()

	fw.size += uint(fw.wc.flushedBytes) - uint(bufSize)
	fw.wc.flushedBytes = 0

	if err != nil {
		err = errors.Unwrap(err)
		return fmt.Errorf(failedToFlushLogBuffer, err)
	}

	return nil
}
