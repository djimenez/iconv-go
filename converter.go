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

func NewConverter(fromEncoding string, toEncoding string) (converter *Converter, err os.Error) {
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
func (this *Converter) Convert(input []byte, output []byte) (bytesRead int, bytesWritten int, err os.Error) {
	inputLeft := C.size_t(len(input))
	outputLeft := C.size_t(len(output))

	// we're going to give iconv the pointers to the underlying
	// storage of each byte slice - so far this is the simplest
	// way i've found to do that in Go, but it seems ugly
	inputFirstElementPointer := &input[0]
	inputPointer := (**C.char)(unsafe.Pointer(&inputFirstElementPointer))

	outputFirstElementPointer := &output[0]
	outputPointer := (**C.char)(unsafe.Pointer(&outputFirstElementPointer))

	// we're only going to make one call to iconv
	if inputLeft > 0 && outputLeft > 0 {
		_,err = C.iconv(this.context, inputPointer, &inputLeft, outputPointer, &outputLeft)

		// update byte counters
		bytesRead = len(input) - int(inputLeft)
		bytesWritten = len(output) - int(outputLeft)
	}
	
	return bytesRead, bytesWritten, err
}

// convert the bytes of a string and return the resulting string
//
// TODO: can we do this in terms of Convert function
func (this *Converter) ConvertString(input string) (output string, err os.Error) {
	// both our input buffer and output buffer will be the same size
	// but we'll reuse our output buffer each time its filled
	bufferSize := len(input)
	sourceLeft := C.size_t(bufferSize)
	outputLeft := sourceLeft
	outputReset := outputLeft

	// our input buffer is the source string, but iconv will track
	// how many bytes has left to process
	sourceBuffer := C.CString(input)
	sourcePointer := &sourceBuffer

	outputBuffer := make([]byte, bufferSize)
	outputFirstPointer := &outputBuffer[0] 
	outputPointer := (**C.char)(unsafe.Pointer(&outputFirstPointer))

	// process the source with iconv in a loop
	for sourceLeft > 0 {
		//fmt.Println("calling to iconv")
		_,err := C.iconv(this.context, sourcePointer, &sourceLeft, outputPointer, &outputLeft)

		//fmt.Println("sourceLeft: ", int(sourceLeft), " outputLeft: ", int(outputLeft))

		// check the err - most interested if we need to expand the output buffer
		if err != nil {
			//fmt.Println("got error value: ", err)

			if err == E2BIG {
				// we need more output buffer to continue
				// instead of resizing, lets pull what we got so far
				// and set outputLeft back to the buffer size
				output += string(outputBuffer[0:bufferSize - int(outputLeft)])
				outputLeft = outputReset
			} else {
				// we got an error we can't continue with
				break
			}
		}
	}

	// free our sourceBuffer, no longer needed
	//C.free(unsafe.Pointer(&sourceBuffer))

	// convert output buffer a go string
	output += string(outputBuffer[0:bufferSize - int(outputLeft)])

	// free our outputBuffer, no longer needed
	//C.free(unsafe.Pointer(&outputBuffer))	
	
	// return result and any err
	return output, err
}
