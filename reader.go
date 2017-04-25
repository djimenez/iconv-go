package iconv

import (
	"io"
	"runtime"
)

const (
	defaultReadBufferSize = 8 * 1024
	minReadBufferSize     = 16
)

type Reader struct {
	source            io.Reader
	converter         *Converter
	buffer            []byte
	readPos, writePos int
	eof               bool
}

func NewReader(source io.Reader, fromEncoding, toEncoding string) (*Reader, error) {
	return NewReaderSized(source, fromEncoding, toEncoding, defaultReadBufferSize)
}

func NewReaderFromConverter(source io.Reader, converter *Converter) *Reader {
	return NewReaderFromConverterSized(source, converter, defaultReadBufferSize)
}

func NewReaderSized(source io.Reader, fromEncoding, toEncoding string, size int) (*Reader, error) {
	converter, err := NewConverter(fromEncoding, toEncoding)

	if err != nil {
		return nil, err
	}

	// add a finalizer for the converter we created
	runtime.SetFinalizer(converter, finalizeConverter)

	return NewReaderFromConverterSized(source, converter, size), nil
}

func NewReaderFromConverterSized(source io.Reader, converter *Converter, size int) *Reader {
	if size < minReadBufferSize {
		size = minReadBufferSize
	}

	return &Reader{
		source:    source,
		converter: converter,
		buffer:    make([]byte, size),
	}
}

func (r *Reader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	var bytesRead, bytesWritten int
	var err error

	// setup for a single read into buffer if possible
	if !r.eof {
		if r.readPos > 0 {
			// slide data to front of buffer
			r.readPos, r.writePos = 0, copy(r.buffer, r.buffer[r.readPos:r.writePos])
		}

		if r.writePos < len(r.buffer) {
			// do the single read
			bytesRead, err = r.source.Read(r.buffer[r.writePos:])

			if bytesRead < 0 {
				panic("iconv: source reader returned negative count from Read")
			}

			r.writePos += bytesRead
			r.eof = err == io.EOF
		}
	}

	if r.readPos < r.writePos || r.eof {
		// convert any buffered data we have, or do a final reset (for shift based conversions)
		bytesRead, bytesWritten, err = r.converter.Convert(r.buffer[r.readPos:r.writePos], p)
		r.readPos += bytesRead

		// if we experienced an iconv error and didn't make progress, report it.
		// if we did make progress, it may be informational only (i.e. reporting
		// an EILSEQ even when using //ignore to skip them)
		if err != nil && bytesWritten == 0 {
			return bytesWritten, err
		}

		// signal an EOF only if we didn't write anything - accomodates premature
		// errror checking in user code
		if bytesWritten == 0 && r.eof {
			return 0, io.EOF
		}

		return bytesWritten, nil
	}

	return 0, err
}

func (r *Reader) Reset(source io.Reader) {
	r.converter.Reset()

	*r = Reader{
		source:    source,
		converter: r.converter,
		buffer:    r.buffer,
	}
}
