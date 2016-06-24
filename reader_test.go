package iconv

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func GbkToUtf8(src []byte) ([]byte, error) {
	reader, err := NewReader(bytes.NewReader(src), "gbk", "utf-8")
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

func Utf8ToGbk(src []byte) ([]byte, error) {
	reader, err := NewReader(bytes.NewReader(src), "utf-8", "gbk")
	reader.buffer = make([]byte, 16)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

func TestReaderWithDataLargerThanBuffer(t *testing.T) {
	chars := []byte("梅")
	for len(chars) < bufferSize*2 {
		t.Logf("input size: %d", len(chars))
		chars = append(chars, chars...)
		_, err := Utf8ToGbk(chars)
		if err != nil {
			t.Fail()
			t.Logf("failed with %d bytes data", len(chars))
			return
		}
	}
}
