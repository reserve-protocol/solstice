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

func main() {
	var contractsDir string
	var txnHash string
	var contractName string
	flag.StringVar(&contractsDir, "contractsDir", "", "directory containing all the contracts")
	flag.StringVar(&txnHash, "txnHash", "0x0", "a transaction hash")
	flag.StringVar(&contractName, "contractName", "", "the full name of a specific contract")
	flag.Parse()

	contractsDir, err := filepath.Abs(contractsDir)
	common.Check(err)

	workingDir, err := os.Getwd()
	common.Check(err)

	if txnHash != "0x0" {
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
		dirName := workingDir + "/" + txnHash

		if _, err := os.Stat(dirName); os.IsNotExist(err) {
		    os.Mkdir(dirName, 0711)
		} else {
			common.Check(err)
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

			common.Check(ioutil.WriteFile(
				dirName + "/" + fmt.Sprintf("%06d", i) + ".html",
				markedUpSource,
				0644,
			))
		}
	} else {
		ast, err := srcmap.GetAST(contractName, contractsDir)
		common.Check(err)
		fmt.Printf("%v\n", ast)
		fmt.Printf("%v\n", ast.Children[3].Children[0].Children[0])

		displayTree(ast)
	}
}

func displayTree(node srcmap.ASTTree) {
	fmt.Printf("%v\n", node.SrcLoc)
	// turn location into OpSourceLocation
	// make a file of this source location
	for _, childNode:= range node.Children {
		displayTree(*childNode)
	}

	return
}
