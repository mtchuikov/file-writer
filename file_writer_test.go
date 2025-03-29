package filewriter

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type testFileWriter struct {
	suite.Suite

	afs *afero.Afero

	fileName    string
	filePayload []byte
	fileSize    uint

	fw *FileWriter
}

func TestFileWriterSuite(t *testing.T) {
	var writer bytes.Buffer
	wc := &writeCounter{wr: &writer}

	fw := &FileWriter{
		Mode:  defaulFileMode,
		Flags: defaulFileFlags,
		Wc:    wc,
		Buf:   bufio.NewWriter(wc),
	}

	filePayload := []byte("Hello, world!\n")
	tf := &testFileWriter{
		afs:         &afero.Afero{Fs: afero.NewMemMapFs()},
		fileName:    "test.log",
		filePayload: filePayload,
		fileSize:    uint(len(filePayload)),
		fw:          fw,
	}

	openFileFn = func(name string, flag int, mode os.FileMode) (file, error) {
		return tf.afs.OpenFile(name, flag, mode)
	}

	renameFileFn = func(oldpath, newpath string) error {
		return tf.afs.Rename(oldpath, newpath)
	}

	suite.Run(t, tf)
}

func (tf *testFileWriter) SetupTest() {
	var writer bytes.Buffer
	tf.fw.Wc = &writeCounter{wr: &writer}
	tf.fw.Buf = bufio.NewWriter(tf.fw.Wc)
}

func (tf *testFileWriter) TearDownSuite() {
	if tf.fw.File != nil {
		tf.fw.File.Close()
	}
}

func (tf *testFileWriter) TestOpen() {
	mode := 0777
	err := tf.fw.Open(tf.fileName, mode)

	msg := "expected no error when oppening file, got '%v'"
	tf.Require().NoError(err, msg, err)

	stat, err := tf.fw.File.Stat()
	msg = "expected no error when getting file stat, got '%v'"
	tf.Require().NoError(err, msg, err)

	tf.Require().Equal(
		os.FileMode(mode), stat.Mode(),
		"expected file mode %v, got '%v'",
		mode, stat.Mode(),
	)
}

func (tf *testFileWriter) TestWrite() {}

func (tf *testFileWriter) TestClose() {}
