package filewriter

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type testUtilsSuite struct {
	suite.Suite

	afs *afero.Afero

	fileName    string
	filePayload []byte
	fileSize    uint

	fw *FileWriter
}

func TestUtilsSuite(t *testing.T) {
	fw := &FileWriter{
		mode:  defaulFileMode,
		flags: defaulFileFlags,
	}

	tu := &testUtilsSuite{
		afs:         &afero.Afero{Fs: afero.NewMemMapFs()},
		fileName:    "test.log",
		filePayload: []byte("Hello, world!\n"),
		fileSize:    14,
		fw:          fw,
	}

	openFileFn = func(name string, flag int, mode os.FileMode) (file, error) {
		return tu.afs.OpenFile(name, flag, mode)
	}

	renameFileFn = func(oldpath, newpath string) error {
		return tu.afs.Rename(oldpath, newpath)
	}

	suite.Run(t, tu)
}

func (tu *testUtilsSuite) TearDownSuite() {
	if tu.fw.file != nil {
		tu.fw.file.Close()
	}
}

func (tu *testUtilsSuite) TestGetOpenFile() {
	tu.afs.WriteFile(tu.fileName, tu.filePayload, defaulFileMode)

	err := tu.fw.openFile(tu.fileName, tu.fw.mode)

	msg := "expected no error when oppening file, got '%v'"
	tu.Require().NoError(err, msg, err)

	tu.Require().Equalf(
		tu.fileSize, tu.fw.size,
		"expected file name to be '%v', got '%v'",
		tu.fileSize, tu.fw.size,
	)
}

func (tu *testUtilsSuite) TestSetBufWriter() {
	var oldWriter bytes.Buffer
	tu.fw.buf = bufio.NewWriter(&oldWriter)

	var newWriter bytes.Buffer
	tu.fw.setBufWriter(&newWriter)

	tu.fw.buf.Write(tu.filePayload)
	tu.fw.buf.Flush()

	tu.Require().Emptyf(
		oldWriter.Bytes(),
		"expected old writer to be empty, got '%v'",
		oldWriter.String(),
	)

	tu.Require().Equal(
		tu.filePayload, newWriter.Bytes(),
		"expected new writer to hold '%v', got '%v'",
		string(tu.filePayload), newWriter.String(),
	)
}

// func (tu *testUtilsSuite) TestRotateFile() {
// 	file, err := openFileFn(tu.fileName, tu.fw.flags, tu.fw.mode)

// 	msg := "expected no error when oppening file, got '%v'"
// 	tu.Require().NoError(err, msg, err)

// 	tu.fw.file = file

// 	var writer bytes.Buffer
// 	wc := &writeCounter{wr: &writer}

// 	tu.fw.wc = wc
// 	tu.fw.buf = bufio.NewWriter(tu.fw.wc)

// 	tu.fw.buf.Write(tu.filePayload)
// 	tu.fw.rotateFile()

// 	exists, err := tu.afs.Exists(tu.fileName + ".")

// 	msg = "expected no error when checking backup file existence, got '%v'"
// 	tu.Require().NoError(err, msg, err)
// 	tu.Require().True(exists, "expected backup file to be exist")

// 	exists, err = tu.afs.Exists(tu.fileName)

// 	msg = "expected no error when checking file existence, got '%v'"
// 	tu.Require().NoError(err, msg, err)
// 	tu.Require().True(exists, "expected file to be exist")
// }

func (tu *testUtilsSuite) TestFlushBuf() {
	tu.fw.size = tu.fileSize

	var writer bytes.Buffer
	wc := &writeCounter{wr: &writer}

	tu.fw.wc = wc
	tu.fw.buf = bufio.NewWriter(tu.fw.wc)

	tu.fw.buf.Write(tu.filePayload)
	err := tu.fw.flushBuf()

	msg := "expected no error when flushing buffer, got '%v'"
	tu.Require().NoError(err, msg, err)

	tu.Require().Equalf(
		tu.fileSize, tu.fw.size,
		"expected file size to be equal to '%v', got '%v'",
		tu.fileSize, tu.fw.size,
	)
}
