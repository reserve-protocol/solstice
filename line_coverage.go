package main

import (
	"context"
	"fmt"
	"math/big"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
    "github.com/spf13/viper"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/srcmap"
	"github.com/coordination-institute/debugging-tools/evmbytecode"
)

func main() {
	common.Check(common.ReadConfig())

	client, err := ethclient.Dial(viper.GetString("blockchain_client"))
	common.Check(err)

	sourceMaps, bytecodeToFilename, err := srcmap.Get()
	common.Check(err)

	headerBeforeTests, err := client.HeaderByNumber(context.Background(), nil)
	common.Check(err)
	fmt.Printf("Start block number: %v\n", headerBeforeTests.Number)

	// Run tests
	{
		args := viper.GetStringSlice("test_command")
		cmd := exec.Command(
			args[0],
			args[1:]...
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Println("Tests return " + fmt.Sprint(err) + ": " + string(output))
			if fmt.Sprint(err) != "exit status 1" || string(output) != "" {
				panic(err)
			}
		}
	}

	blockAfterTests, err := client.BlockByNumber(context.Background(), nil)
	common.Check(err)
	fmt.Printf("Ending block number: %v\n", blockAfterTests.Number())

	// Build list of all transactions
	txns := []*types.Transaction{}

	for blockNumber := headerBeforeTests.Number; blockNumber.Cmp(blockAfterTests.Number()) < 0; blockNumber.Add(blockNumber, big.NewInt(1)) {
		var oneMore big.Int
		oneMore.Add(blockNumber, big.NewInt(1))
		block, err := client.BlockByNumber(context.Background(), &oneMore)
		common.Check(err)
		for _, txn := range block.Transactions() {
			if txn.To() != nil {
				bytecode, err := client.CodeAt(context.TODO(), *txn.To(), block.Number())
				common.Check(err)
				// If it's a function call and not just an ETH txn
				if len(bytecode) != 0 {
					txns = append(txns, txn)
				}
			}
		}
	}

	// We have a list of contract names, but we need a list of file names; there can be many contracts per file.
	sourceFileName := []string{}
	{
		sourceFileNameSet := make(map[string]struct{})
		i := 0
		for contractName := range sourceMaps {
			filename := strings.Split(contractName, ":")[0]

			if _, ok := sourceFileNameSet[filename]; !ok {
				sourceFileNameSet[filename] = struct{}{}
				sourceFileName = append(sourceFileName, filename)
			}

			i++
		}
	}

	// Initialize the coverage report
	coverage := make(map[string][]int)
	for _, sourceFileName := range sourceFileName {
		lineLength, err := common.NumberOfLines(sourceFileName)
		common.Check(err)
		coverage[sourceFileName] = make([]int, lineLength)
	}

	// Fill the coverage report
	for _, txn := range txns {
		execTrace, err := parity.GetExecTrace(fmt.Sprintf("0x%x", txn.Hash()))
		common.Check(err)
		pcToOpIndex := evmbytecode.GetPcToOpIndex(execTrace.Code)
		contractName := bytecodeToFilename[common.RemoveMetaData(execTrace.Code)]
		if contractName == "" {
			continue
		}
		for _, traceOp := range execTrace.Ops {
			location := sourceMaps[contractName][pcToOpIndex[traceOp.Pc]]
			if location.ByteLength == -1 || location.ByteOffset == -1 || location.SourceFileName == "" {
				continue
			}

			lineNumber, _, _, err := location.ByteLocToSnippet()
			common.Check(err)

			coverage[location.SourceFileName][lineNumber] += 1
		}
	}

	// Print the coverage report
	for filename, lines := range coverage {
		fmt.Println(filename)
		for linenumber, count := range lines {
			if count != 0 {
				fmt.Printf("Line %d has %d hits\n", linenumber, count)
			}
		}
	}
}
