package iconv

import (
	"io"
	"runtime"
	"syscall"
)

const (
	defaultWriteBufferSize = 8 * 1024
	minWriteBufferSize     = 16
)

type Writer struct {
	destination             io.Writer
	converter               *Converter
	readBuffer, writeBuffer []byte
	readPos, writePos       int
}

func NewWriter(destination io.Writer, fromEncoding string, toEncoding string) (*Writer, error) {
	return NewWriterSized(destination, fromEncoding, toEncoding, defaultWriteBufferSize)
}

func NewWriterFromConverter(destination io.Writer, converter *Converter) (writer *Writer) {
	return NewWriterFromConverterSized(destination, converter, defaultWriteBufferSize)
}

func NewWriterSized(destination io.Writer, fromEncoding, toEncoding string, size int) (*Writer, error) {
	converter, err := NewConverter(fromEncoding, toEncoding)

	if err != nil {
		return nil, err
	}

	// add a finalizer for the converter we created
	runtime.SetFinalizer(converter, finalizeConverter)

	return NewWriterFromConverterSized(destination, converter, size), nil
}

func NewWriterFromConverterSized(destination io.Writer, converter *Converter, size int) *Writer {
	if size < minWriteBufferSize {
		size = minWriteBufferSize
	}

	return &Writer{
		destination: destination,
		converter:   converter,
		readBuffer:  make([]byte, size),
		writeBuffer: make([]byte, size),
	}
}

// Implements io.Writer
//
// Will attempt to convert all of p into buffer. If there's not enough room in
// the buffer to hold all converted bytes, the buffer will be flushed and p will
// continue to be processed. Close should be called on a writer when finished
// with all writes, to ensure final shift sequences are written and buffer is
// flushed to underlying io.Writer.
//
// Can return all errors that Convert can, as well as any errors from Flush. Note
// that some errors from Convert are suppressed if we continue making progress
// on p.
func (w *Writer) Write(p []byte) (int, error) {
	var totalBytesRead, bytesRead, bytesWritten int
	var err error

	if w.readPos == 0 || len(p) == 0 {
		bytesRead, bytesWritten, err = w.converter.Convert(p, w.writeBuffer[w.writePos:])
		totalBytesRead += bytesRead
		w.writePos += bytesWritten
		w.readPos = 0
	} else {
		// we have left over bytes from previous write that weren't complete and there's at least
		// one byte being written, fill read buffer with p and try to convert, if we make progress
		// we can continue conversion from p itself
		bytesCopied := copy(w.readBuffer[w.readPos:], p)

		bytesRead, bytesWritten, err = w.converter.Convert(w.readBuffer[:w.readPos+bytesCopied], w.writeBuffer[w.writePos:])

		// if we made no progress, give up
		if bytesRead <= w.readPos {
			return 0, err
		}

		bytesRead -= w.readPos
		totalBytesRead += bytesRead

		w.readPos = 0
		w.writePos += bytesWritten
	}

	// try to process all of p - lots of io functions don't like short writes.
	//
	// There are a few error cases we need to treat specially, as long as we've
	// made progress on p, E2BIG and EILSEQ should not be fatal. EINVAL isn't
	// fatal as long as the rest of p fits in our buffers.
	for err != nil && bytesRead > 0 {
		switch err {
		case syscall.E2BIG:
			err = w.Flush()

		case syscall.EILSEQ:
			// IGNORE suffix still reports the error on convert
			err = nil

			// if no more bytes, don't do an empty convert (resets state)
			if totalBytesRead == len(p) {
				break
			}

		case syscall.EINVAL:
			// if the rest of p fits in read buffer copy it there
			if len(p[totalBytesRead:]) <= len(w.readBuffer) {
				w.readPos = copy(w.readBuffer, p[totalBytesRead:])
				totalBytesRead += w.readPos
				break
			}
		}

		// if not an ignoreable err or Flush err
		if err != nil {
			break
		}

		bytesRead, bytesWritten, err = w.converter.Convert(p[totalBytesRead:], w.writeBuffer[w.writePos:])
		totalBytesRead += bytesRead
		w.writePos += bytesWritten
	}

	return totalBytesRead, err
}

// Attempt to write any buffered data to destination writer. Returns error from
// Write call or io.ErrShortWrite if Write didn't report an error but also didn't
// accept all bytes given.
func (w *Writer) Flush() error {
	if w.readPos < w.writePos {
		bytesWritten, err := w.destination.Write(w.writeBuffer[:w.writePos])

		if bytesWritten < 0 {
			panic("iconv: writer returned negative count from Write")
		}

		if bytesWritten > 0 {
			w.writePos = copy(w.writeBuffer, w.writeBuffer[bytesWritten:w.writePos])
		}

		if err == nil && w.writePos > 0 {
			err = io.ErrShortWrite
		}

		return err
	}

	return nil
}

// Perform a final write with empty buffer, which allows iconv to close any shift
// sequences. A Flush is performed if needed.
func (w *Writer) Close() error {
	_, err := w.Write(nil)
	if err != nil {
		return err
	}
	return w.Flush()
}

// Reset state and direct writes to a new destination writer
func (w *Writer) Reset(destination io.Writer) {
	w.converter.Reset()

	*w = Writer{
		destination: destination,
		converter:   w.converter,
		readBuffer:  w.readBuffer,
		writeBuffer: w.writeBuffer,
	}
}
