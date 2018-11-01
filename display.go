package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/srcmap"
	"github.com/coordination-institute/debugging-tools/trace"
)

func main() {
	var txnHash string
	var contractsDir string
	var pcIndex int
	flag.StringVar(&txnHash, "txnHash", "0x0", "a transaction hash")
	flag.StringVar(&contractsDir, "contractsDir", "", "directory containing all the contracts")
	flag.IntVar(&pcIndex, "pcIndex", -1, "Chosen index into exec trace")
	flag.Parse()

	contractsDir, err := filepath.Abs(contractsDir)
	common.Check(err)

	execTrace, err := parity.GetExecTrace(txnHash)
	common.Check(err)

	sourceMaps, bytecodeToFilename, err := srcmap.Get(contractsDir)
	common.Check(err)

	filename := bytecodeToFilename[common.RemoveMetaData(execTrace.Code)]
	sourceMap := sourceMaps[filename]
	if len(sourceMap) == 0 {
		fmt.Println("Contract code not in contracts dir.")
		return
	}

	workingDir, err := os.Getwd()
	common.Check(err)
	txnDir := workingDir + "/" + txnHash

	if _, err := os.Stat(txnDir); os.IsNotExist(err) {
	    os.Mkdir(txnDir, 0711)
	} else {
		common.Check(err)
	}

	pcToOpIndex := trace.GetPcToOpIndex(execTrace.Code)

	if pcIndex == -1 {
		var prevLoc srcmap.OpSourceLocation
		for i, _ := range execTrace.Ops {
			pc := execTrace.Ops[i].Pc

			if _, ok := pcToOpIndex[pc]; !ok {
				fmt.Println("Something has gone wrong")
				continue
			}

			nextLoc := sourceMap[pcToOpIndex[pc]]

			if nextLoc.SourceFileName == "" {
				continue
			}

			if nextLoc.ByteOffset == prevLoc.ByteOffset &&
				nextLoc.ByteLength == prevLoc.ByteLength &&
				nextLoc.SourceFileName == prevLoc.SourceFileName {
				continue
			} else {
				prevLoc = nextLoc
			}

			markedUpSource, err := nextLoc.LocationMarkup()
			if err != nil {
				continue
			}

			common.Check(ioutil.WriteFile(
				txnDir + "/" + fmt.Sprintf("%06d", i) + ".html",
				markedUpSource,
				0644,
			))
		}
	} else {
		pc := execTrace.Ops[pcIndex].Pc

		if _, ok := pcToOpIndex[pc]; !ok {
			fmt.Println("Something has gone wrong")
			return
		}

		markedUpSource, err := sourceMap[pcToOpIndex[pc]].LocationMarkup()
		common.Check(err)

		common.Check(ioutil.WriteFile(
			txnDir + "/" + strconv.Itoa(pcIndex) + ".html",
			markedUpSource,
			0644,
		))
	}
}
