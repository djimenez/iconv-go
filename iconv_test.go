package iconv

import (
	"bytes"
	"io"
	"strings"
	"syscall"
	"testing"
)

type iconvTest struct {
	description    string
	input          string
	inputEncoding  string
	output         string
	outputEncoding string
	bytesRead      int
	bytesWritten   int
	convertErr     error // err from Convert (raw iconv)
	err            error // err from CovertString, Reader, Writer
}

var (
	iconvTests = []iconvTest{
		iconvTest{
			"simple utf-8 to latin1 conversion success",
			"Hello World!", "utf-8",
			"Hello World!", "latin1",
			12, 12, nil, nil,
		},
		iconvTest{
			"invalid source encoding causes EINVAL",
			"", "doesnotexist",
			"", "utf-8",
			0, 0, syscall.EINVAL, syscall.EINVAL,
		},
		iconvTest{
			"invalid destination encoding causes EINVAL",
			"", "utf-8",
			"", "doesnotexist",
			0, 0, syscall.EINVAL, syscall.EINVAL,
		},
		iconvTest{
			"utf-8 to utf-8 passthrough",
			"Hello world!", "utf-8",
			"Hello world!", "utf-8",
			12, 12, nil, nil,
		},
		iconvTest{
			"utf-8 to utf-8 partial",
			"Hello\xFFWorld!", "utf-8",
			"Hello", "utf-8",
			5, 5, syscall.EILSEQ, syscall.EILSEQ,
		},
		iconvTest{
			"utf-8 to utf-8 ignored",
			"Hello \xFFWorld!", "utf-8",
			"Hello World!", "utf-8//IGNORE",
			13, 12, syscall.EILSEQ, nil,
		},
		iconvTest{
			"invalid input sequence causes EILSEQ",
			"\xFF", "utf-8",
			"", "latin1",
			0, 0, syscall.EILSEQ, syscall.EILSEQ,
		},
		iconvTest{
			"incomplete input sequence causes EINVAL",
			"\xC2", "utf-8",
			"", "latin1",
			0, 0, syscall.EINVAL, syscall.EINVAL,
		},
		iconvTest{
			"invalid input causes partial output and EILSEQ",
			"Hello\xFF", "utf-8",
			"Hello", "latin1",
			5, 5, syscall.EILSEQ, syscall.EILSEQ,
		},
		iconvTest{
			"incomplete input causes partial output and EILSEQ",
			"Hello\xC2", "utf-8",
			"Hello", "latin1",
			5, 5, syscall.EINVAL, syscall.EINVAL,
		},
		/* this is only true for glibc / iconv
		iconvTest{
			"valid input but no conversion causes EILSEQ",
			"你好世界 Hello World", "utf-8",
			"", "latin1",
			0, 0, syscall.EILSEQ, syscall.EILSEQ,
		},*/
		iconvTest{
			"invalid input with ignore",
			"Hello\xFF World!", "utf-8",
			"Hello World!", "latin1//IGNORE",
			13, 12, syscall.EILSEQ, nil,
		},
		iconvTest{
			"valid input but no conversion with IGNORE",
			"你好世界 Hello World", "utf-8",
			" Hello World", "latin1//IGNORE",
			24, 12, syscall.EILSEQ, nil,
		},
		iconvTest{
			"valid input but no conversion with TRANSLIT",
			"你好世界 Hello World", "utf-8",
			"???? Hello World", "latin1//TRANSLIT",
			24, 16, nil, nil,
		},
	}

	ignoreDetected, translitDetected bool
)

func init() {
	// detect if IGNORE / TRANSLIT is supported (glic / libiconv)
	conv, err := NewConverter("utf-8", "ascii//IGNORE")

	if err == nil {
		ignoreDetected = true
		conv.Close()
	}

	conv, err = NewConverter("utf-8", "ascii//TRANSLIT")

	if err == nil {
		translitDetected = true
		conv.Close()
	}
}

func runTests(t *testing.T, f func(iconvTest, *testing.T) (int, int, string, error)) {
	for _, test := range iconvTests {
		t.Run(test.description, func(t *testing.T) {
			if !ignoreDetected && strings.HasSuffix(test.outputEncoding, "//IGNORE") {
				t.Skip("//IGNORE not supported")
			}

			if !translitDetected && strings.HasSuffix(test.outputEncoding, "//TRANSLIT") {
				t.Skip("//TRANSLIT not supported")
			}

			bytesRead, bytesWritten, output, err := f(test, t)

			// check that bytesRead is same as expected
			if bytesRead != test.bytesRead {
				t.Errorf("bytesRead: %d expected: %d", bytesRead, test.bytesRead)
			}

			// check that bytesWritten is same as expected
			if bytesWritten != test.bytesWritten {
				t.Errorf("bytesWritten: %d expected: %d", bytesWritten, test.bytesWritten)
			}

			// check output bytes against expected
			if output != test.output {
				t.Errorf("output: %x expected: %x", output, test.output)
			}

			// check that err is same as expected
			if err != test.err {
				if test.err != nil {
					if err != nil {
						t.Errorf("err: %q expected: %q", err, test.err)
					} else {
						t.Errorf("err: nil expected %q", test.err)
					}
				} else {
					t.Errorf("unexpected error: %q", err)
				}
			}
		})
	}
}

func TestConvert(t *testing.T) {
	runTests(t, func(test iconvTest, t *testing.T) (int, int, string, error) {
		input := []byte(test.input)
		output := make([]byte, 50)

		// peform the conversion
		bytesRead, bytesWritten, err := Convert(input, output, test.inputEncoding, test.outputEncoding)

		// HACK Convert has different erorrs, so check ourselves, and then fake out later check
		if err != test.convertErr {
			if test.convertErr != nil {
				if err != nil {
					t.Errorf("err: %q expected: %q", err, test.convertErr)
				} else {
					t.Errorf("err: nil expected %q", test.convertErr)
				}
			} else {
				t.Errorf("unexpected error: %q", err)
			}
		}
		err = test.err

		return bytesRead, bytesWritten, string(output[:bytesWritten]), err
	})
}

func TestConvertString(t *testing.T) {
	runTests(t, func(test iconvTest, t *testing.T) (int, int, string, error) {
		// perform the conversion
		output, err := ConvertString(test.input, test.inputEncoding, test.outputEncoding)

		// bytesRead and bytesWritten are spoofed a little
		return test.bytesRead, len(output), output, err
	})
}

func TestReader(t *testing.T) {
	runTests(t, func(test iconvTest, t *testing.T) (int, int, string, error) {
		var bytesRead, bytesWritten, finalBytesWritten int
		var err error

		input := bytes.NewBufferString(test.input)
		output := make([]byte, 50)

		reader, err := NewReader(input, test.inputEncoding, test.outputEncoding)

		if err == nil {
			bytesWritten, err = reader.Read(output)

			// we can compute how many bytes iconv read by inspecting the reader state
			bytesRead = len([]byte(test.input)) - input.Len() - (reader.writePos - reader.readPos)

			// with current tests and buffer sizes, we'd expect all input to be buffered if we called read
			if input.Len() != 0 {
				t.Error("not all bytes from input were buffered")
			}

			// do final read test if we can - either get EOF or same test error
			if err == nil {
				finalBytesWritten, err = reader.Read(output[bytesWritten:])

				if finalBytesWritten != 0 {
					t.Errorf("finalBytesWritten: %d expected: 0", finalBytesWritten)
				}

				if err == io.EOF {
					err = nil
				}
			}
		}

		return bytesRead, bytesWritten, string(output[:bytesWritten]), err
	})
}

func TestWriter(t *testing.T) {
	runTests(t, func(test iconvTest, t *testing.T) (int, int, string, error) {
		var bytesRead, bytesWritten int
		var err error

		input := []byte(test.input)
		output := new(bytes.Buffer)

		writer, err := NewWriter(output, test.inputEncoding, test.outputEncoding)

		if err == nil {
			bytesRead, err = writer.Write(input)
			bytesRead -= writer.readPos
			writer.Close()

			bytesWritten = output.Len()
		}

		return bytesRead, bytesWritten, output.String(), err
	})
}

func TestReaderWithCopy(t *testing.T) {
	runTests(t, func(test iconvTest, t *testing.T) (int, int, string, error) {
		input := bytes.NewBufferString(test.input)
		output := new(bytes.Buffer)

		reader, err := NewReader(input, test.inputEncoding, test.outputEncoding)

		if err == nil {
			_, err := io.Copy(output, reader)

			bytesRead := len(test.input) - input.Len() - reader.writePos
			bytesWritten := output.Len()

			return bytesRead, bytesWritten, output.String(), err
		}

		return 0, 0, output.String(), err
	})
}

func TestWriterWithCopy(t *testing.T) {
	runTests(t, func(test iconvTest, t *testing.T) (int, int, string, error) {
		input := bytes.NewBufferString(test.input)
		output := new(bytes.Buffer)

		writer, err := NewWriter(output, test.inputEncoding, test.outputEncoding)

		if err == nil {
			bytesCopied, err := io.Copy(writer, input)
			bytesRead := int(bytesCopied) - writer.readPos
			writer.Close()

			bytesWritten := output.Len()

			return bytesRead, bytesWritten, output.String(), err
		}

		return 0, 0, output.String(), err
	})
}

func TestReaderMultipleReads(t *testing.T) {
	// setup a source reader and our expected output string
	source := bytes.NewBufferString("\x80\x8A\x99\x95\x8B\x86\x87")
	expected := "€Š™•‹†‡"

	// setup reader - use our minimum buffer size so we can force it to shuffle the buffer around
	reader, err := NewReaderSized(source, "cp1252", "utf-8", minReadBufferSize)

	if err != nil {
		if err == syscall.EINVAL {
			t.Skip("Either cp1252 or utf-8 isn't supported by iconv on your system")
		} else {
			t.Fatalf("Unexpected error when creating reader: %s", err)
		}
	}

	// setup a read buffer - we'll slice it to different sizes in our tests
	buffer := make([]byte, 64)

	// first read should fill internal buffer, but we'll only read part of it
	bytesRead, err := reader.Read(buffer[:5])

	if bytesRead != 5 || err != nil {
		t.Fatalf("first read did not give expected 5, nil: %d, %s", bytesRead, err)
	}

	// because of how small teh source is and our minimum buffer size, source shoudl be fully read
	if source.Len() != 0 {
		t.Fatalf("first read did not buffer all of source like expected: %d bytes remain", source.Len())
	}

	// Buffer doesn't return EOF with last bytes, reader shouldn't know its EOF yet
	if reader.eof {
		t.Fatalf("first read was not expected to receive EOF")
	}

	// second read should shift internal buffer, and fill again - make buffer too small for last utf-8 character
	// E2BIG from iconv should be ignored because we wrote at least 1 byte
	bytesRead, err = reader.Read(buffer[5:18])

	if bytesRead != 12 || err != nil {
		t.Fatalf("second read did not give expected 15, nil: %d, %s", bytesRead, err)
	}

	if !reader.eof {
		t.Fatalf("second read did not put reader into eof state")
	}

	// try to read the last 3 byte character with only a buffer of 2 bytes - this time we should see the E2BIG
	bytesRead, err = reader.Read(buffer[17:19])

	if bytesRead != 0 || err != syscall.E2BIG {
		t.Fatalf("third read did not give expected 0, E2BIG: %d, %s", bytesRead, err)
	}

	// fourth read should finish last character
	bytesRead, err = reader.Read(buffer[17:])

	if bytesRead != 3 || err != nil {
		t.Fatalf("fourth read did not give expected 3, nil: %d, %s", bytesRead, err)
	}

	// last read should be EOF
	bytesRead, err = reader.Read(buffer[20:])

	if bytesRead != 0 || err != io.EOF {
		t.Fatalf("final read did not give expected 0, EOF: %d, %s", bytesRead, err)
	}

	// check full utf-8 output
	if string(buffer[:20]) != expected {
		t.Fatalf("output did not match expected %q: %q", expected, string(buffer[:20]))
	}
}

func TestWriteWithIncompleteSequence(t *testing.T) {
	expected := "\x80\x8A\x99\x95\x8B\x86\x87"
	input := []byte("€Š™•‹†‡")
	output := new(bytes.Buffer)

	writer, err := NewWriter(output, "utf-8", "cp1252")

	if err != nil {
		t.Fatalf("unexpected error while creating writer %q", err)
	}

	// the input string is made of 3 byte characters, for the test we want to only write part of the last character
	bytesFromBuffer := len(input) - 2

	bytesRead, err := writer.Write(input[:bytesFromBuffer])

	if bytesRead != bytesFromBuffer {
		t.Fatalf("did a short write on first write: %d, %s", bytesRead, err)
	}

	// finish the rest
	bytesRead, err = writer.Write(input[bytesFromBuffer:])

	if bytesRead != 2 {
		t.Fatalf("did a short write on second write: %d, %s", bytesRead, err)
	}

	err = writer.Close()
	actual := output.String()

	if err != nil {
		t.Errorf("got an error on close: %s", err)
	}

	if actual != expected {
		t.Errorf("output %x did not match expected %x", actual, expected)
	}
}

func TestWriteWithIncompleteSequenceAndIgnore(t *testing.T) {
	if !ignoreDetected {
		t.Skip("//IGNORE not supported")
	}

	expected := "\x80\x8A\x99\x95\x8B\x86\x87"
	input := []byte("€Š™•‹†‡")
	output := new(bytes.Buffer)

	writer, err := NewWriter(output, "utf-8", "cp1252//IGNORE")

	if err != nil {
		t.Fatalf("unexpected error while creating writer %q", err)
	}

	// the input string is made of 3 byte characters, for the test we want to only write part of the last character
	bytesFromBuffer := len(input) - 2

	bytesRead, err := writer.Write(input[:bytesFromBuffer])

	if bytesRead != bytesFromBuffer {
		t.Fatalf("did a short write on first write: %d, %s", bytesRead, err)
	}

	// finish the rest
	bytesRead, err = writer.Write(input[bytesFromBuffer:])

	if bytesRead != 2 {
		t.Fatalf("did a short write on second write: %d, %s", bytesRead, err)
	}

	err = writer.Close()
	actual := output.String()

	if err != nil {
		t.Errorf("got an error on close: %s", err)
	}

	if actual != expected {
		t.Errorf("output %x did not match expected %x", actual, expected)
	}
}

func TestWriteWithIncompleteSequenceAtEOF(t *testing.T) {
	expected := "\x80\x8A\x99\x95\x8B\x86"
	input := []byte("€Š™•‹†‡")
	output := new(bytes.Buffer)

	writer, err := NewWriter(output, "utf-8", "cp1252")

	if err != nil {
		t.Fatalf("unexpected error while creating writer %q", err)
	}

	// the input string is made of 3 byte characters, for the test we want to only write part of the last character
	bytesFromBuffer := len(input) - 2

	bytesRead, err := writer.Write(input[:bytesFromBuffer])

	if bytesRead != bytesFromBuffer {
		t.Fatalf("did a short write on first write: %d, %s", bytesRead, err)
	}

	err = writer.Close()
	actual := output.String()

	if err != nil {
		t.Errorf("got an error on close: %s", err)
	}

	if actual != expected {
		t.Errorf("output %x did not match expected %x", actual, expected)
	}
}
