package iconv

// #include <iconv.h>
import "C"

import (
	"os"
	"unsafe"
)

type Converter struct {
	context C.iconv_t
	open bool
}

func NewConverter(fromEncoding string, toEncoding string) (converter *Converter, err Error) {
	converter = new(Converter)

	converter.context, err = C.iconv_open(C.CString(toEncoding), C.CString(fromEncoding))

	// check err
	if err == nil {
		// no error, mark the context as open
		converter.open = true
	}

	return
}

// Called before garbage collection
func (this *Converter) destroy() {
	this.Close()
}

// The converter can be explicitly closed if desired
func (this *Converter) Close() (err os.Error) {
	if this.open {
		_, err = C.iconv_close(this.context)
	}

	return
}

// read bytes from an input buffer, and write them to and output buffer
// will return the number of bytesRead from the input and the number of bytes
// written to the output as well as any iconv errors
//
// NOTE: not all bytes may be consumed from the input. This can be because the output
// buffer is too small or because there were iconv errors
func (this *Converter) Convert(input []byte, output []byte) (bytesRead int, bytesWritten int, err Error) {
	inputLeft := C.size_t(len(input))
	outputLeft := C.size_t(len(output))
	
	if inputLeft > 0 && outputLeft > 0 {
		// we're going to give iconv the pointers to the underlying
		// storage of each byte slice - so far this is the simplest
		// way i've found to do that in Go, but it seems ugly
		inputFirstElementPointer := &input[0]
		inputPointer := (**C.char)(unsafe.Pointer(&inputFirstElementPointer))

		outputFirstElementPointer := &output[0]
		outputPointer := (**C.char)(unsafe.Pointer(&outputFirstElementPointer))

		// we're only going to make one call to iconv
		_,err = C.iconv(this.context, inputPointer, &inputLeft, outputPointer, &outputLeft)

		// update byte counters
		bytesRead = len(input) - int(inputLeft)
		bytesWritten = len(output) - int(outputLeft)
	}
	
	return bytesRead, bytesWritten, err
}

// convert a string value, returning a new string value
func (this *Converter) ConvertString(input string) (output string, err Error) {

	// construct the buffers
	inputBuffer := []byte(input)
	outputBuffer := make([]byte, len(inputBuffer) * 2) // we use a larger buffer to help avoid resizing later

	// call Convert until all input bytes are read or an error occurs
	var bytesRead, totalBytesRead, bytesWritten, totalBytesWritten int

	for totalBytesRead < len(inputBuffer) && err == nil {
		bytesRead, bytesWritten, err = this.Convert(inputBuffer, outputBuffer)

		totalBytesRead += bytesRead
		totalBytesWritten += bytesWritten

		// check for the E2BIG error specifically, we can add to the output
		// buffer to correct for it and then continue
		if err == E2BIG {
			// increase the size of the output buffer by another input length
			// first, create a new buffer
			tempBuffer := make([]byte, len(outputBuffer) + len(inputBuffer))
			
			// copy the existing data
			copy(tempBuffer, outputBuffer)

			// switch the buffers
			outputBuffer = tempBuffer

			// forget the error
			err = nil
		}
	}

	// construct the final output string
	output = string(outputBuffer[:totalBytesWritten])

	return output, err
}
