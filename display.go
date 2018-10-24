package main

import (
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/source_map"
	"github.com/coordination-institute/debugging-tools/trace"
)

const githubGreen string = "#e6ffed"

func main() {
	var txnHash string
	var contractsDir string
	var pcIndex int
	flag.StringVar(&txnHash, "txnHash", "0x0", "a transaction hash")
	flag.StringVar(&contractsDir, "contractsDir", "", "directory containing all the contracts")
	flag.IntVar(&pcIndex, "pcIndex", 0, "Chosen index into exec trace")
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

	pc := execTrace.Ops[pcIndex].Pc
	fmt.Printf("Op index: %v\n", pcToOpIndex[pc])

	// Now you have pcToOpIndex[pc] with which to pick an operation from the source map

	sourceMaps, bytecodeToFilename, err := source_map.GetSourceMaps(contractsDir)
	common.Check(err)

	filename := bytecodeToFilename[execTrace.Code[0:len(execTrace.Code)-86]]
	srcmap := sourceMaps[filename]
	if len(srcmap) == 0 {
		fmt.Println("Contract code not in contracts dir.")
		return
	}

	if _, ok := pcToOpIndex[pc]; !ok {
		fmt.Println("Something has gone wrong")
		return
	}
	lastLocation := srcmap[pcToOpIndex[pc]]

	if lastLocation.SourceFileName == "" {
		fmt.Printf("File name: %s\n has no source map.\n", filename)
		return
	}
	fmt.Printf("Last location: {%d %d %s %c}\n", lastLocation.ByteOffset, lastLocation.ByteLength, lastLocation.SourceFileName, lastLocation.JumpType)

	wholeSrc, err := ioutil.ReadFile(lastLocation.SourceFileName)
	common.Check(err)

	srcBeginning := html.EscapeString(string(wholeSrc[0:lastLocation.ByteOffset]))
	srcMiddle := html.EscapeString(string(wholeSrc[lastLocation.ByteOffset : lastLocation.ByteOffset+lastLocation.ByteLength]))
	srcEnd := html.EscapeString(string(wholeSrc[lastLocation.ByteOffset+lastLocation.ByteLength : len(wholeSrc)-1]))

	coverageFilename := "/home/altair/go/src/github.com/coordination-institute/debugging-tools/coverage.html"
	coverageFile, err := os.Open(coverageFilename)
	common.Check(err)
	defer coverageFile.Close()

	markedUpSource := []byte("<pre>" +
		srcBeginning +
		"<div style=\"background-color:" + githubGreen + ";\">" +
		srcMiddle +
		"</div>" +
		srcEnd +
		"</pre>",
	)

	common.Check(ioutil.WriteFile(coverageFilename, markedUpSource, 0644))
}
