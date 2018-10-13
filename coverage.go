// TODO: Things to make configurable; the contract directory, the client port number

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/source_map"
	"github.com/coordination-institute/debugging-tools/trace"
)

func main() {
	var contractsDir string
	flag.StringVar(&contractsDir, "contractsDir", "", "the directory containing all the contracts")
	flag.Parse()

	contractsDir, err := filepath.Abs(contractsDir)
	if err != nil {
		panic(err)
	}

	// TODO: Make this port etc configurable
	client, err := ethclient.Dial("http://127.0.0.1:8545")
	if err != nil {
		panic(err)
	}


	sourceMaps, bytecodeToFilename, err := source_map.GetSourceMaps(contractsDir)
	if err != nil {
		panic(err)
	}


	headerBeforeTests, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Start block number: %v\n", headerBeforeTests.Number)

	// Run tests
	{
		// TODO: Make this configurable
		cmd := exec.Command(
			"go",
			"test",
			filepath.Join(
				"./github.com/coordination-institute/reserve/",
				"./protocol/system_tests/",
			),
			"--tags",
			"ethereum",
			"--count",
			"1",
		)

		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
		    fmt.Println("Tests return " + fmt.Sprint(err) + ": " + stderr.String())
			if fmt.Sprint(err) != "exit status 1" || stderr.String() != "" {
		    	panic(stderr.String())
			}
		}
	}

	blockAfterTests, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Ending block number: %v\n", blockAfterTests.Number())

	// Build list of all transactions
	txns := []*types.Transaction{}

	for blockNumber := headerBeforeTests.Number; blockNumber.Cmp(blockAfterTests.Number()) < 0; blockNumber.Add(blockNumber, big.NewInt(1)) {
		var oneMore big.Int
		oneMore.Add(blockNumber, big.NewInt(1))
		block, err := client.BlockByNumber(context.Background(), &oneMore)
		if err != nil {
			panic(err)
		}
		for _, txn := range block.Transactions() {
			if txn.To() != nil {
				bytecode, err := client.CodeAt(context.TODO(), *txn.To(), block.Number())
				if err != nil {
					panic(err)
				}
				if len(bytecode) != 0 {
					txns = append(txns, txn)
				}
			}
		}
	}

	sourceFileNameSet   := make(map[string]struct{})
	sourceFileNameSlice := []string{}

	i := 0
	for contractName := range sourceMaps {
		filename := strings.Split(contractName, ":")[0]

		if _, ok := sourceFileNameSet[filename]; !ok {
		  sourceFileNameSet[filename] = struct{}{}
		  sourceFileNameSlice = append(sourceFileNameSlice, filename)
		}

	    i++
	}

	// Initialize the coverage report
	coverage := make(map[string][]int)
	for _, sourceFileName := range sourceFileNameSlice {
		lineLength, err := lineCounter(sourceFileName)
		if err != nil {
			panic(err)
		}
		coverage[sourceFileName] = make([]int, lineLength)
	}

	// Fill the coverage report
	for _, txn := range txns {
		execTrace, err := parity.GetExecTrace(fmt.Sprintf("0x%x", txn.Hash()))
		if err != nil {
			panic(err)
		}
		pcToOpIndex := trace.GetPcToOpIndex(execTrace)
		for _, traceOp := range execTrace.Ops {
			contractName := bytecodeToFilename[execTrace.Code[0:len(execTrace.Code)-86]]
			if contractName == "" {
				continue
			}
			lastLocation := sourceMaps[contractName][pcToOpIndex[traceOp.Pc]]
			if lastLocation.ByteLength == -1 || lastLocation.ByteOffset == -1 || lastLocation.SourceFileName == "" {
				continue
			}

			sourceFileReader, err := os.Open(lastLocation.SourceFileName)
			if err != nil {
				panic(err)
			}
			sourceFileBeginning := make([]byte, lastLocation.ByteOffset+lastLocation.ByteLength)

			_, err = io.ReadFull(sourceFileReader, sourceFileBeginning)
			if err != nil {
				panic(err)
			}

			lineNumber := 1
			columnNumber := 1
			for byteIndex, sourceByte := range sourceFileBeginning {
				if byteIndex < lastLocation.ByteOffset {
					columnNumber += 1
					if sourceByte == '\n' {
						lineNumber += 1
						columnNumber = 1
					}
				}
			}
			coverage[lastLocation.SourceFileName][lineNumber] += 1
			sourceFileReader.Close()
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

func lineCounter(filename string) (int, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := reader.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
