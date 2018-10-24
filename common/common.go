package common

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/coordination-institute/debugging-tools/source_map"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func NumberOfLines(filename string) (int, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	return bytes.Count(b, []byte{'\n'}), nil
}

func ByteLocToSnippet(location source_map.OpSourceLocation) (int, int, []byte, error) {
	sourceFileReader, err := os.Open(location.SourceFileName)
	if err != nil {
		return 0, 0, []byte{}, err
	}
	sourceFileBeginning := make([]byte, location.ByteOffset+location.ByteLength)

	_, err = io.ReadFull(sourceFileReader, sourceFileBeginning)
	if err != nil {
		return 0, 0, []byte{}, err
	}
	defer sourceFileReader.Close()

	lineNumber := 1
	columnNumber := 1
	var codeSnippet []byte
	for byteIndex, sourceByte := range sourceFileBeginning {
		if byteIndex < location.ByteOffset {
			columnNumber += 1
			if sourceByte == '\n' {
				lineNumber += 1
				columnNumber = 1
			}
		} else {
			codeSnippet = append(codeSnippet, sourceByte)
		}
	}
	return lineNumber, columnNumber, codeSnippet, nil
}
