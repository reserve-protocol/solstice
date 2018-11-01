package main

import (
	"errors"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/srcmap"
	"github.com/coordination-institute/debugging-tools/trace"
)

const githubGreen string = "#e6ffed"

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
	srcmap := sourceMaps[filename]
	if len(srcmap) == 0 {
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
		var prevLoc common.OpSourceLocation
		for i, _ := range execTrace.Ops {
			pc := execTrace.Ops[i].Pc

			if _, ok := pcToOpIndex[pc]; !ok {
				fmt.Println("Something has gone wrong")
				continue
			}

			nextLoc := srcmap[pcToOpIndex[pc]]

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

			markedUpSource, err := displayStep(nextLoc)
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

		markedUpSource, err := displayStep(srcmap[pcToOpIndex[pc]])
		common.Check(err)

		common.Check(ioutil.WriteFile(
			txnDir + "/" + strconv.Itoa(pcIndex) + ".html",
			markedUpSource,
			0644,
		))
	}
}

func displayStep(location common.OpSourceLocation) ([]byte, error) {
	if location.SourceFileName == "" {
		return []byte{}, errors.New("Step source file not found.")
	}

	fmt.Printf("Location: {%d %d %s %c}\n", location.ByteOffset, location.ByteLength, location.SourceFileName, location.JumpType)

	wholeSrc, err := ioutil.ReadFile(location.SourceFileName)
	if err != nil {
		return []byte{}, err
	}

	srcBeginning := html.EscapeString(string(wholeSrc[0:location.ByteOffset]))
	srcMiddle := html.EscapeString(string(wholeSrc[location.ByteOffset : location.ByteOffset+location.ByteLength]))
	srcEnd := html.EscapeString(string(wholeSrc[location.ByteOffset+location.ByteLength : len(wholeSrc)-1]))

	return []byte("<pre>" +
		srcBeginning +
		"<span style=\"background-color:" + githubGreen + ";\">" +
		srcMiddle +
		"</span>" +
		srcEnd +
		"</pre>",
	), nil
}
