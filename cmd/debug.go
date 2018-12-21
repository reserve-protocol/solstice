package cmd

import (
	"fmt"

    "github.com/spf13/cobra"

	"github.com/reserve-protocol/solstice/common"
	"github.com/reserve-protocol/solstice/evmbytecode"
	"github.com/reserve-protocol/solstice/parity"
	"github.com/reserve-protocol/solstice/srcmap"
)

func init() {
	debugCmd.Flags().StringVar(&txnHash, "txn", "0x0", "a transaction hash to debug")
	debugCmd.MarkFlagRequired("txn")
	rootCmd.AddCommand(debugCmd)
}

var debugCmd = &cobra.Command{
    Use:   "debug",
    Short: "Tells you where a txn reverted",
    Long: `Tells you the last line of code that a particular transaction ended 
on. This is especially useful for reverts, since the EVM does not currently 
provide any kind of error messages or stack traces.`,
    Run: Debug,
}

func Debug(cmd *cobra.Command, args []string) {
	execTrace, err := parity.GetExecTrace(txnHash)
	common.Check(err)
	if execTrace.Code == "0x" {
		fmt.Println("Transaction was not sent to a contract.")
		return
	}

	pcToOpIndex := evmbytecode.GetPcToOpIndex(execTrace.Code)

	lastProgramCounter := execTrace.Ops[len(execTrace.Ops)-1].PC
	fmt.Printf("Last program counter: %v\n", lastProgramCounter)
	fmt.Printf("Final op index: %v\n", pcToOpIndex[lastProgramCounter])

	// Now you have pcToOpIndex[lastProgramCounter] with which to pick an operation from the source map

	sourceMaps, bytecodeToFilename, err := srcmap.Get()
	common.Check(err)

	filename := bytecodeToFilename[evmbytecode.RemoveMetaData(execTrace.Code)]
	sourceMap := sourceMaps[filename]
	if len(sourceMap) == 0 {
		fmt.Println("Contract code not in contracts dir.")
		return
	}

	if _, ok := pcToOpIndex[lastProgramCounter]; !ok {
		fmt.Println("Something has gone wrong")
		return
	}
	lastLocation := sourceMap[pcToOpIndex[lastProgramCounter]]

	if lastLocation.SourceFileName == "" {
		fmt.Printf("File name:\n%s\nhas no source map.", filename)
		return
	}

	lineNumber, columnNumber, codeSnippet, err := lastLocation.ByteLocToSnippet()
	common.Check(err)

	fmt.Printf("%s %d:%d\n", lastLocation.SourceFileName, lineNumber, columnNumber)
	fmt.Printf("... %s ...\n", codeSnippet)
}
