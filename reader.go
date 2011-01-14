package iconv

import ( 
	"io"
	"os"
)

type Reader struct {
	source io.Reader
	converter *Converter
	rawBuffer []byte
	rawReadPos, rawWritePos int
	convertedBuffer []byte
	convertedReadPos, convertedWritePos int
}

func NewReader(source io.Reader, fromEncoding string, toEncoding string) (*Reader, os.Error) {
	// create a converter
	converter, err := NewConverter(fromEncoding, toEncoding)

	if err == nil {
		return NewReaderFromConverter(source, converter), err
	}

	// return the error
	return nil, err
}

func NewReaderFromConverter(source io.Reader, converter *Converter) (reader *Reader) {
	reader = new(Reader)

	// copy elements
	reader.source = source
	reader.converter = converter

	// create 8K buffers
	reader.rawBuffer = make([]byte, 8 * 1024)
	reader.convertedBuffer = make([]byte, 8 * 1024)

	return reader
}

func (this *Reader) fillRawBuffer() {
	// slide existing data to beginning
	if this.rawReadPos > 0 {
		// copy current bytes
		copy(this.rawBuffer, this.rawBuffer[this.rawReadPos:this.rawWritePos])

		// adjust positions
		this.rawWritePos -= this.rawReadPos
		this.rawReadPos = 0
	}

	// read new data into buffer at write position
	bytesRead, err := this.source.Read(this.rawBuffer[this.rawWritePos:])

	// adjust write position
	this.rawWritePos += bytesRead

	// track source reader errors
	if err != nil {
		// not sure where to put this for now
	}
}

func (this *Reader) fillConvertedBuffer() {
	// slide existing data to beginning
	if this.convertedReadPos > 0 {
		// copy current bytes
		copy(this.convertedBuffer, this.convertedBuffer[this.convertedReadPos:this.convertedWritePos])

		// adjust positions
		this.convertedWritePos -= this.convertedReadPos
		this.convertedReadPos = 0
	}

	// use iconv to fill the converted buffer from the raw buffer
	bytesRead, bytesWritten, err := this.converter.Convert(this.rawBuffer[this.rawReadPos:this.rawWritePos], this.convertedBuffer[this.convertedWritePos:])

	// adjust read and write positions
	this.rawReadPos += bytesRead
	this.convertedWritePos += bytesWritten

	// track iconv convert errors
	if err != nil {
		// not sure where to put this for now
	}
}

// implement the io.Reader interface
func (this *Reader) Read(p []byte) (n int, err os.Error) {
	this.fillRawBuffer()
	this.fillConvertedBuffer()

	if this.convertedWritePos - 1 > this.convertedReadPos {
		// copy converted bytes into p
		n = copy(p, this.convertedBuffer[this.convertedReadPos:this.convertedWritePos])
	}

	return
}
