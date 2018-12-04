package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"

	"github.com/coordination-institute/debugging-tools/ast"
	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/evmbytecode"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/srclocation"
	"github.com/coordination-institute/debugging-tools/srcmap"
)

var dirName string

func main() {
	common.Check(common.ReadConfig())

	var txnHash string
	var contractName string
	flag.StringVar(&txnHash, "txnHash", "", "a transaction hash")
	flag.StringVar(&contractName, "contractName", "", "the name of a specific contract")
	flag.Parse()

	workingDir, err := os.Getwd()
	common.Check(err)

	if txnHash != "" {
		dirName = workingDir + "/" + txnHash
	} else {
		dirName = workingDir + "/" + contractName
	}

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		os.MkdirAll(dirName, 0711)
	} else {
		common.Check(err)
	}

	if txnHash != "" {
		execTrace, err := parity.GetExecTrace(txnHash)
		common.Check(err)

		sourceMaps, bytecodeToFilename, err := srcmap.Get()
		common.Check(err)

		filename := bytecodeToFilename[evmbytecode.RemoveMetaData(execTrace.Code)]
		sourceMap := sourceMaps[filename]
		if len(sourceMap) == 0 {
			fmt.Println("Contract code not in contracts dir.")
			return
		}

		pcToOpIndex := evmbytecode.GetPcToOpIndex(execTrace.Code)

		var prevLoc srclocation.SourceLocation
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

			// It's not currently useful to display duplicate execution steps
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

			writeLocFile(markedUpSource, uint(i))
		}
	} else {
		ast, err := ast.Get(viper.GetString("contracts_dir") + "/" + contractName)
		common.Check(err)
		displayTree(ast)
	}
}

func displayTree(node ast.ASTTree) {
	markedUpSource, err := node.SrcLoc.LocationMarkup()
	common.Check(err)

	writeLocFile(markedUpSource, node.Id)

	for _, childNode := range node.Children {
		displayTree(*childNode)
	}

	return
}

func writeLocFile(contents []byte, index uint) {
	common.Check(ioutil.WriteFile(
		dirName+"/"+fmt.Sprintf("%06d", index)+".html",
		contents,
		0644,
	))
	return
}
