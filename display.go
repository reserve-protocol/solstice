package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/srcmap"
	"github.com/coordination-institute/debugging-tools/trace"
)

var dirName string

func main() {
	var contractsDir string
	var txnHash string
	var contractName string
	flag.StringVar(&contractsDir, "contractsDir", "", "directory containing all the contracts")
	flag.StringVar(&txnHash, "txnHash", "", "a transaction hash")
	flag.StringVar(&contractName, "contractName", "", "the full name of a specific contract")
	flag.Parse()

	contractsDir, err := filepath.Abs(contractsDir)
	common.Check(err)

	workingDir, err := os.Getwd()
	common.Check(err)

	if txnHash != "" {
		dirName = workingDir + "/" + txnHash
	} else {
		dirName = workingDir + "/" + contractName
	}

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
	    os.Mkdir(dirName, 0711)
	} else {
		common.Check(err)
	}

	if txnHash != "" {
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

		pcToOpIndex := trace.GetPcToOpIndex(execTrace.Code)

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

			writeLocFile(markedUpSource, i)
		}
	} else {
		ast, err := srcmap.GetAST(contractName, contractsDir)
		common.Check(err)
		displayTree(ast)
	}
}

func displayTree(node srcmap.ASTTree) {
	markedUpSource, err := node.SrcLoc.LocationMarkup()
	common.Check(err)

	writeLocFile(markedUpSource, node.Id)

	for _, childNode:= range node.Children {
		displayTree(*childNode)
	}

	return
}

func writeLocFile(contents []byte, index int) {
	common.Check(ioutil.WriteFile(
		dirName + "/" + fmt.Sprintf("%06d", index) + ".html",
		contents,
		0644,
	))
	return
}
