package iconv

/*
#include <errno.h>
*/
import "C"

import (
	"os"
)

// allows us to check for iconv specific errors
type Error os.Error

var (
	EILSEQ Error = os.Errno(int(C.EILSEQ))
	E2BIG Error = os.Errno(int(C.E2BIG))
)

func Convert(input []byte, output []byte, fromEncoding string, toEncoding string) (bytesRead int, bytesWritten int, err Error) {
	// create a new converter
	converter, err := NewConverter(fromEncoding, toEncoding)

	if err == nil {
		// call Convert
		bytesRead, bytesWritten, err = converter.Convert(input, output)

		// close the converter
		converter.Close()
	}

	return
}

func ConvertString(input string, fromEncoding string, toEncoding string) (output string, err Error) {
	// create a new converter
	converter, err := NewConverter(fromEncoding, toEncoding)

	if err == nil {
		// convert the string
		output, err = converter.ConvertString(input)

		// close the converter
		converter.Close()
	}

	return
}
