package main

import (
	"context"
	"fmt"
	"html"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
    "github.com/spf13/viper"

	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/parity"
	"github.com/coordination-institute/debugging-tools/srcmap"
	"github.com/coordination-institute/debugging-tools/trace"
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
	coverageMap := make(map[string][]srcmap.CoverageLoc)
	for _, sourceFileName := range sourceFileName {
		coverageLocs, err := srcmap.GetCoverageLocs(sourceFileName)
		common.Check(err)
		coverageMap[sourceFileName] = coverageLocs
	}

	// Fill the coverage report
	for _, txn := range txns {
		execTrace, err := parity.GetExecTrace(fmt.Sprintf("0x%x", txn.Hash()))
		common.Check(err)
		pcToOpIndex := trace.GetPcToOpIndex(execTrace.Code)
		contractName := bytecodeToFilename[common.RemoveMetaData(execTrace.Code)]
		if contractName == "" {
			continue
		}
		for _, traceOp := range execTrace.Ops {
			traceLoc := sourceMaps[contractName][pcToOpIndex[traceOp.Pc]]
			if traceLoc.ByteLength == -1 || traceLoc.ByteOffset == -1 || traceLoc.SourceFileName == "" {
				continue
			}
			if traceLoc.ByteLength == 0 {
				continue
			}

			for i, coverageLoc := range coverageMap[traceLoc.SourceFileName] {
				if coverageLoc.CoverageRange.ByteLength == traceLoc.ByteLength &&
					coverageLoc.CoverageRange.ByteOffset == traceLoc.ByteOffset {
					coverageMap[traceLoc.SourceFileName][i].HitCount += 1
				}
			}
		}
	}

	// I'm printing it as a sanity check
	// for file, locs := range coverageMap {
	// 	fmt.Printf("File: %v\n", file)
	// 	fmt.Println("Locs:")
	// 	for _, loc := range locs {
	// 		fmt.Printf("    SrcLoc: %v, %v, %v\n", loc.CoverageRange.ByteOffset, loc.CoverageRange.ByteLength, loc.HitCount)
	// 		if len(loc.SrcLocs) == 1 {
	// 			continue
	// 		}
	// 		for _, piece := range loc.SrcLocs {
	// 			fmt.Printf("        %v, %v\n", piece.ByteOffset, piece.ByteLength)
	// 		}
	// 	}
	// }

	// Write the coverage report
	for filename, locs := range coverageMap {
		origSource, err := ioutil.ReadFile(filename)
		common.Check(err)

		var flatLocs []coverageCount
		for _, covLoc := range locs {
			for _, loc := range covLoc.SrcLocs {
				if loc.ByteLength == 0 {
					continue
				}

				flatLocs = append(flatLocs, coverageCount{
					loc,
					covLoc.HitCount,
				})
			}
		}

		sortedLocs := byByteOffset(flatLocs)
		sort.Sort(sortedLocs)

		markedUpString := "<pre>"
		markupIndex := 0

		for _, covCountLoc := range sortedLocs {
			if covCountLoc.SrcLoc.ByteLength == 0 {
				continue
			}

			// if covCountLoc.SrcLoc.ByteOffset < markupIndex {
			// 	continue
			// }
			markedUpString += html.EscapeString(string(origSource[markupIndex : covCountLoc.SrcLoc.ByteOffset]))
			if covCountLoc.count == 0 {
				markedUpString += "<span style=\"background-color:" + srcmap.GithubRed + ";\">"
			} else {
				markedUpString += "<span style=\"background-color:" + srcmap.GithubGreen + ";\">"
			}
			// if covCountLoc.SrcLoc.ByteOffset + covCountLoc.SrcLoc.ByteLength > len(origSource) ||
			//     covCountLoc.SrcLoc.ByteOffset > len(origSource) || 
			//     covCountLoc.SrcLoc.ByteOffset > covCountLoc.SrcLoc.ByteOffset + covCountLoc.SrcLoc.ByteLength {
			// 	fmt.Printf("length: %v\n", len(origSource))
			// 	fmt.Printf("covCountLoc.SrcLoc: %v\n", covCountLoc.SrcLoc)
			// }
			markedUpString += html.EscapeString(string(origSource[covCountLoc.SrcLoc.ByteOffset : covCountLoc.SrcLoc.ByteOffset + covCountLoc.SrcLoc.ByteLength]))
			markedUpString += "</span>"
			markupIndex = covCountLoc.SrcLoc.ByteOffset + covCountLoc.SrcLoc.ByteLength
		}

		markedUpString += "</pre>"
		markedUpSource := []byte(markedUpString)

		relativeFileName := strings.TrimPrefix(filename, viper.GetString("contracts_dir"))
		reportFileName := viper.GetString("coverage_report_dir") + relativeFileName + ".html"

		if _, err := os.Stat(filepath.Dir(reportFileName)); os.IsNotExist(err) {
		    common.Check(os.MkdirAll(filepath.Dir(reportFileName), 0711))
		} else {
			common.Check(err)
		}

		common.Check(ioutil.WriteFile(reportFileName, markedUpSource, 0644))
	}
}

type coverageCount struct {
	SrcLoc srcmap.OpSourceLocation
	count  int
}

// ~~~~~~~ Sorting OpSourceLocations ~~~~~~~
type byByteOffset []coverageCount

func (ls byByteOffset) Len() int {
    return len(ls)
}

func (ls byByteOffset) Swap(i, j int) {
    ls[i], ls[j] = ls[j], ls[i]
}

func (ls byByteOffset) Less(i, j int) bool {
	// If they have the same byte offset, my current belief is that that can
	// only happen if one of them is empty. In that case, we're going to throw
	// it away anyway.
    return ls[i].SrcLoc.ByteOffset < ls[j].SrcLoc.ByteOffset
}
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
