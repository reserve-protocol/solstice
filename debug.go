// TODO: Things to make configurable; the contract directory, the client port number

package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/source_map"
	"github.com/coordination-institute/debugging-tools/trace"
)

func main() {
	var txnHash string
	var contractsDir string
	flag.StringVar(&txnHash, "txnHash", "0x0", "a transaction hash")
	flag.StringVar(&contractsDir, "contractsDir", "", "directory containing all the contracts")
	flag.Parse()

	contractsDir, err := filepath.Abs(contractsDir)
	common.Check(err)

	execTrace, err := parity.GetExecTrace(txnHash)
	common.Check(err)
	if execTrace.Code == "0x" {
		fmt.Println("Transaction was not sent to a contract.")
		return
	}

	pcToOpIndex := trace.GetPcToOpIndex(execTrace.Code)

	lastProgramCounter := execTrace.Ops[len(execTrace.Ops)-1].Pc
	fmt.Printf("Last program counter: %v\n", lastProgramCounter)
	fmt.Printf("Final op index: %v\n", pcToOpIndex[lastProgramCounter])

	// Now you have pcToOpIndex[lastProgramCounter] with which to pick an operation from the source map

	sourceMaps, bytecodeToFilename, err := source_map.GetSourceMaps(contractsDir)
	common.Check(err)

	filename := bytecodeToFilename[execTrace.Code[0:len(execTrace.Code)-86]]
	srcmap := sourceMaps[filename]
	if len(srcmap) == 0 {
		fmt.Println("Contract code not in contracts dir.")
		return
	}

	if _, ok := pcToOpIndex[lastProgramCounter]; !ok {
		fmt.Println("Something has gone wrong")
		return
	}
	lastLocation := srcmap[pcToOpIndex[lastProgramCounter]]

	if lastLocation.SourceFileName == "" {
		fmt.Printf("File name: %s\n has no source map.", filename)
		return
	}
	fmt.Printf("Last location: {%d %d %s %c}\n", lastLocation.ByteOffset, lastLocation.ByteLength, lastLocation.SourceFileName, lastLocation.JumpType)

	lineNumber, columnNumber, codeSnippet, err := common.ByteLocToSnippet(lastLocation)
	common.Check(err)

	fmt.Printf("%s %d:%d\n", lastLocation.SourceFileName, lineNumber, columnNumber)
	fmt.Printf("... %s ...\n", codeSnippet)
}
