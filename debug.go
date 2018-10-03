// TODO: Things to make configurable; the contract directory, the client port number

package main

import (
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

	trace := execTrace.Ops
	lastProgramCounter := trace[len(trace)-1].Pc
	fmt.Printf("Last program counter: %v\n", lastProgramCounter)
	fmt.Printf("Final op index: %v\n", pcToOpIndex[lastProgramCounter])

	// Now you have pcToOpIndex[lastProgramCounter] with which to pick an operation from the source map

	opSourceLocations, sourceFileList, err := source_map.GetSourceMap(sourceFilePath, contractsPath)
	if err != nil {
		panic(err)
	}
	lastLocation := opSourceLocations[pcToOpIndex[lastProgramCounter]]
	fmt.Printf("Last location: {%d %d %d %c}\n", lastLocation.ByteOffset, lastLocation.ByteLength, lastLocation.SourceFileIndex, lastLocation.JumpType)

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
	var codeSnippet []byte
	for byteIndex, sourceByte := range sourceFileBeginning {
		if byteIndex < lastLocation.ByteOffset {
			columnNumber += 1
			if sourceByte == '\n' {
				lineNumber += 1
				columnNumber = 1
			}
		} else {
			codeSnippet = append(codeSnippet, sourceByte)
		}
	}

	fmt.Printf("%s %d:%d\n", sourceFileName, lineNumber, columnNumber)
	fmt.Printf("... %s ...\n", codeSnippet)
}
