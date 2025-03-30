# file-writer

file-writer provides a wrapper for writing logs to a file. It writes data to an in-memory buffer and then writes to disk in batches, which helps optimize performance. Additionally, the library allows for log file rotation based on size, compressing older files using the gzip algorithm.

TODO:
- [ ] Cover the code with tests
- [ ] Add more examples
- [x] Add compression of log files using gzip
- [ ] Add flushing of the log buffer using a time.Ticker

## Example of usage

file-writer can be integrated with popular logging libraries. This example shows how to use it with zerolog to log messages to stdout and to a file:

```go
package main

import (
	"os"

	fw "github.com/mtchuikov/file-writer"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	fw, err := fw.NewFileWriter("test.log")
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("failed to setup file writer")
	}

	output := zerolog.MultiLevelWriter(os.Stdout, fw)
	logger = logger.Output(output)

	logger.Info().
		Msg("Hello, world!")
}
```


