// TODO: Things to make configurable; the contract directory, the client port number

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/source_map"
	"github.com/coordination-institute/debugging-tools/trace"
)

func main() {
	var txnHash string
	var sourceFilePath string
	var contractName string
	flag.StringVar(&txnHash, "txnHash", "0x0", "a transaction hash")
	flag.StringVar(&sourceFilePath, "sourceFilePath", "", "contract file path")
	flag.StringVar(&contractName, "contractName", "", "contract file name")
	flag.Parse()

	if contractName == "" {
		// If they don't specify a contract name, assume it's the same as the filename.
		contractName = strings.Split(filepath.Base(sourceFilePath), ".")[0]
	}
	contractsPath := filepath.Join(
		build.Default.GOPATH,
		"/src/github.com/coordination-institute/reserve/protocol/ethereum/contracts",
	)

	execTrace, err := parity.GetExecTrace(txnHash)
	if err != nil {
		panic(err)
	}

	pcToOpIndex := trace.GetPcToOpIndex(execTrace)

	opSourceLocations, sourceFileList, err := source_map.GetSourceMap(sourceFilePath, contractsPath)
	if err != nil {
		panic(err)
	}

	coverage := make(map[string][]int)
	for _, sourceFile := range sourceFileList {
		sourceFileName := filepath.Join(contractsPath, sourceFile)
		lineLength, err := lineCounter(sourceFileName)
		if err != nil {
			panic(err)
		}
		coverage[sourceFileName] = make([]int, lineLength)
	}

	for _, traceOp := range execTrace.Ops {
		lastLocation := opSourceLocations[pcToOpIndex[traceOp.Pc]]
		if lastLocation.ByteLength == -1 || lastLocation.ByteOffset == -1 || lastLocation.SourceFileIndex == -1 {
			continue
		}

		sourceFileName := filepath.Join(contractsPath, sourceFileList[lastLocation.SourceFileIndex])
		sourceFileReader, err := os.Open(sourceFileName)
		if err != nil {
			panic(err)
		}
		defer sourceFileReader.Close()
		sourceFileBeginning := make([]byte, lastLocation.ByteOffset+lastLocation.ByteLength)

		_, err = io.ReadFull(sourceFileReader, sourceFileBeginning)
		if err != nil {
			panic(err)
		}

		lineNumber := 1
		columnNumber := 1
		for byteIndex, sourceByte := range sourceFileBeginning {
			if byteIndex < lastLocation.ByteOffset {
				columnNumber += 1
				if sourceByte == '\n' {
					lineNumber += 1
					columnNumber = 1
				}
			}
		}
		coverage[sourceFileName][lineNumber] += 1
	}

	for filename, lines := range coverage {
		fmt.Println(filename)
		for linenumber, count := range lines {
			if count != 0 {
				fmt.Printf("Line %d has %d hits\n", linenumber, count)
			}
		}
	}
}

func lineCounter(filename string) (int, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := reader.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
