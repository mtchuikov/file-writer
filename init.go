package filewriter

// bufWriterFieldOffset is the byte offset within the internal
// bufio.Writer structure where the underlying io.Writer interface
// is stored.
//
// When rotating a log file, this offset is used to directly access
// the private 'wr' field (of type io.Writer) in the bufio.Writer
// structure, thereby avoiding the need to allocate a new structure
// and copy data from the old buffer into a new one.
var bufWriterFieldOffset uintptr

func init() {
	// On a 64-bit system, ^uintptr(0) converted to uint64 equals
	// ^uint64(0), so if is64Bit is true, we set bufWriterFieldOffset
	// to 48; otherwise, on a 32-bit system, we set it to 24.
	//
	// This method of determining the processor architecture was proposed by
	// Karl on StackOverflow https://stackoverflow.com/a/60319709
	is64Bit := uint64(^uintptr(0)) == ^uint64(0)
	if is64Bit {
		bufWriterFieldOffset = 48
		return
	}

	bufWriterFieldOffset = 24
}
