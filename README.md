Install
=======

The goinstall command can be used:

	goinstall github.com/djimenez/iconv.go

Or, you can clone the repository and use gomake instead

	git clone git://github.com/djimenez/iconv.go.git iconv
	cd iconv
	gomake install

Usage
=====

To use the package, you'll need the appropriate import statement:

	import (
		// if you used goinstall, you'll want this import
		iconv "github.com/djimenez/iconv.go"

		// if you used gomake install directly, you'll want this import
		iconv
	)

Converting string Values 
------------------------

Converting a string can be done with two methods. First, there's iconv.ConvertString(input, fromEncoding, toEncoding string)

	output,_ := iconv.ConvertString("Hello World!", "utf-8", "windows-1252")

Alternatively, you can create a converter and use its ConvertString method. This mostly just saves having to parse the from and to encodings when converting many strings in the same way.

	converter := iconv.NewConverter("utf-8", "windows-1252")
	output,_ := converter.ConvertString("Hello World!")

Converting []byte Values
------------------------

Converting a []byte can similarly be done with two methods. First, there's iconv.Convert(input, output []byte, fromEncoding, toEncoding string). You'll immediately notice this requires you to give it both the input and output buffer. Ideally, the output buffer should be sized so that it can hold all converted bytes from input, but if it cannot, then Convert will put as many bytes as it can into the buffer without creating an invalid sequence. For example, if iconv only has a single byte left in the output buffer but needs 2 or more for the complete character in a multibyte encoding it will stop writing to the buffer and return with an iconv.E2BIG error.

	input := []byte("Hello World!")
	output := make([]byte, len(input))
	
	bytesRead, bytesWritten, error := iconv.Convert(input, output, "utf-8", "windows-1252")

Just like with ConvertString, there is also a Convert method on Converter that can be used.

	...
	converter := iconv.NewConverter("utf-8", "windows-1252")
	
	bytesRead, bytesWritten, error := converter.Convert(input, output)

Converting an *io.Reader
------------------------

The iconv.Reader allows any other *io.Reader to be wrapped and have its bytes transcoded as they are read. 

	// We're wrapping stdin for simplicity, but a File or network reader could be wrapped as well
	reader,_ := iconv.NewReader(os.Stdin, "utf-8", "windows-1252")

Converting an *io.Writer
------------------------

To be written.

Piping a Conversion
-------------------

To be written.
