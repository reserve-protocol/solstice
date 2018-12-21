package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

    "github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/reserve-protocol/solstice/ast"
	"github.com/reserve-protocol/solstice/common"
	"github.com/reserve-protocol/solstice/evmbytecode"
	"github.com/reserve-protocol/solstice/parity"
	"github.com/reserve-protocol/solstice/srclocation"
	"github.com/reserve-protocol/solstice/srcmap"
)

var contractName string

func init() {
	displayCmd.Flags().StringVar(&txnHash, "txn", "", "the hash of a transaction to display the trace of")
	displayCmd.Flags().StringVar(&contractName, "contract", "", "the name of a contract to display the AST of")
	rootCmd.AddCommand(displayCmd)
}

var displayCmd = &cobra.Command{
    Use:   "display",
    Short: "Prints marked up source code of the requested info",
    Long: `If a transaction ID is given, it delivers marked up source code for 
each step in the transaction, similar to a stack trace. If given a contract 
file name, it delivers marked up source code for each node in the abstract 
syntax tree (AST) of that file.`,
    Run: Display,
}

var dirName string

func Display(cmd *cobra.Command, args []string) {
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
			pc := execTrace.Ops[i].PC

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

func displayTree(node ast.AST) {
	markedUpSource, err := node.SrcLoc.LocationMarkup()
	common.Check(err)

	writeLocFile(markedUpSource, node.ID)

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
