package cmd

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
    "github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/reserve-protocol/solstice/ast"
	"github.com/reserve-protocol/solstice/common"
	"github.com/reserve-protocol/solstice/covloc"
	"github.com/reserve-protocol/solstice/evmbytecode"
	"github.com/reserve-protocol/solstice/parity"
	"github.com/reserve-protocol/solstice/srclocation"
	"github.com/reserve-protocol/solstice/srcmap"
)

func init() {
	rootCmd.AddCommand(coverCmd)
}

var coverCmd = &cobra.Command{
    Use:   "cover",
    Short: "Generates code coverage report",
    Long: `Generates code coverage report.`,
    Run: Cover,
}

func Cover(cmd *cobra.Command, args []string) {
	client, err := ethclient.Dial(viper.GetString("blockchain_client"))
	common.Check(err)

	sourceMaps, bytecodeToFilename, err := srcmap.Get()
	common.Check(err)

	ctx := context.Background()
	headerBeforeTests, err := client.HeaderByNumber(ctx, nil)
	common.Check(err)
	fmt.Printf("Start block number: %v\n", headerBeforeTests.Number)

	// Run tests
	{
		args := viper.GetStringSlice("test_command")
		cmd := exec.Command(
			args[0],
			args[1:]...,
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Tests return %v: %s\n", err, output)
			if err.Error() != "exit status 1" || string(output) != "" {
				panic(err)
			}
		}
	}

	blockAfterTests, err := client.BlockByNumber(ctx, nil)
	common.Check(err)
	fmt.Printf("Ending block number: %v\n", blockAfterTests.Number())

	// Build list of all transactions
	var txns []*types.Transaction
	for blockNumber := headerBeforeTests.Number; blockNumber.Cmp(blockAfterTests.Number()) < 0; blockNumber.Add(blockNumber, big.NewInt(1)) {
		var oneMore big.Int
		oneMore.Add(blockNumber, big.NewInt(1))
		block, err := client.BlockByNumber(ctx, &oneMore)
		common.Check(err)
		for _, txn := range block.Transactions() {
			if txn.To() != nil {
				bytecode, err := client.CodeAt(ctx, *txn.To(), block.Number())
				common.Check(err)
				// If it's a function call and not just an ETH txn
				if len(bytecode) != 0 {
					txns = append(txns, txn)
				}
			}
		}
	}

	// We have a list of contract names, but we need a list of file names; there can be many contracts per file.
	var sourceFileName []string
	{
		sourceFileNameSet := make(map[string]struct{})
		for contractName := range sourceMaps {
			filename := strings.Split(contractName, ":")[0]

			if _, ok := sourceFileNameSet[filename]; !ok {
				sourceFileNameSet[filename] = struct{}{}
				sourceFileName = append(sourceFileName, filename)
			}
		}
	}

	// Initialize the coverage report
	coverageMap := make(map[string][]covloc.CoverageLoc)

	plainASTs, err := ast.FromSrcmaps()
	common.Check(err)
	for sourceFileName, plainAST := range plainASTs {
		coverageLocs, err := covloc.ToCoverageLocs(plainAST)
		common.Check(err)
		coverageMap[sourceFileName] = coverageLocs
	}

	// Fill the coverage report
	for _, txn := range txns {
		execTrace, err := parity.GetExecTrace(fmt.Sprintf("0x%x", txn.Hash()))
		common.Check(err)
		pcToOpIndex := evmbytecode.GetPcToOpIndex(execTrace.Code)
		contractName := bytecodeToFilename[evmbytecode.RemoveMetaData(execTrace.Code)]
		if contractName == "" {
			continue
		}
		for _, traceOp := range execTrace.Ops {
			traceLoc := sourceMaps[contractName][pcToOpIndex[traceOp.PC]]
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

	// Write the coverage report
	for filename, locs := range coverageMap {
		origSource, err := ioutil.ReadFile(filename)
		common.Check(err)

		var flatLocs []covloc.CoverageCount
		for _, covLoc := range locs {
			for _, loc := range covLoc.SrcLocs {
				if loc.ByteLength == 0 {
					continue
				}

				flatLocs = append(flatLocs, covloc.CoverageCount{
					loc,
					covLoc.HitCount,
				})
			}
		}

		sort.Slice(flatLocs, func(i, j int) bool {
			// If they have the same byte offset, my current belief is that that can
			// only happen if one of them is empty. In that case, we're going to throw
			// it away anyway.
			return flatLocs[i].SrcLoc.ByteOffset < flatLocs[j].SrcLoc.ByteOffset
		})

		markedUpString := "<pre>"
		markupIndex := 0

		for _, covCountLoc := range flatLocs {
			markedUpString += html.EscapeString(string(origSource[markupIndex:covCountLoc.SrcLoc.ByteOffset]))
			if covCountLoc.Count == 0 {
				markedUpString += "<span style=\"background-color:" + srclocation.GithubRed + ";\">"
			} else {
				markedUpString += "<span style=\"background-color:" + srclocation.GithubGreen + ";\">"
			}
			markedUpString += html.EscapeString(string(origSource[covCountLoc.SrcLoc.ByteOffset : covCountLoc.SrcLoc.ByteOffset+covCountLoc.SrcLoc.ByteLength]))
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
