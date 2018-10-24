package common

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type OpSourceLocation struct {
	ByteOffset     int
	ByteLength     int
	SourceFileName string
	JumpType       rune
}

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

func ByteLocToSnippet(location OpSourceLocation) (int, int, []byte, error) {
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

func RemoveMetaData(bytecode string) string {
	if len(bytecode) < 2 || !strings.HasPrefix(bytecode, "0x") {
		panic(errors.New("Bytecode must start with 0x."))
	}

	if bytecode == "0x" {
		return bytecode
	}

	if len(bytecode) < 86 {
		panic(errors.New("Bytecode does not have metadata. Maybe it's an old version, or maybe it's been removed already."))
	}

	metadata := bytecode[len(bytecode)-86 : len(bytecode)]

	if !strings.HasPrefix(metadata, "a165627a7a72305820") {
		panic(errors.New("Metadata prefix malformed or absent."))
	}

	if !strings.HasSuffix(metadata, "0029") {
		panic(errors.New("Metadata postfix malformed or absent."))
	}

	return bytecode[0 : len(bytecode)-86]
}
