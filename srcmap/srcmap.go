package srcmap

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/coordination-institute/debugging-tools/ast"
	"github.com/coordination-institute/debugging-tools/common"
	"github.com/coordination-institute/debugging-tools/evmbytecode"
	"github.com/coordination-institute/debugging-tools/solc"
	"github.com/coordination-institute/debugging-tools/srclocation"
)

func Get() (map[string][]srclocation.SourceLocation, map[string]string, error) {
	files, err := common.AllContracts()
	if err != nil {
		return map[string][]srclocation.SourceLocation{}, map[string]string{}, err
	}

	srcMapJSON, err := solc.GetCombinedJSON("srcmap-runtime,bin-runtime", files)
	if err != nil {
		return map[string][]srclocation.SourceLocation{}, map[string]string{}, err
	}

	bytecodeToFilename := make(map[string]string)
	for contractName, artifacts := range srcMapJSON.Contracts {
		if len(artifacts.BinRuntime) != 0 {
			bytecode := "0x" + artifacts.BinRuntime
			bytecodeToFilename[evmbytecode.RemoveMetaData(bytecode)] = contractName
		}
	}

	sourceMaps := map[string][]srclocation.SourceLocation{}
	for contractName, artifacts := range srcMapJSON.Contracts {
		sourceMaps[contractName], err = Decompress(artifacts.SrcmapRuntime, srcMapJSON.SourceList)
		if err != nil {
			return sourceMaps, bytecodeToFilename, err
		}
	}
	return sourceMaps, bytecodeToFilename, err
}

func Decompress(srcMap string, srcList []string) ([]srclocation.SourceLocation, error) {
	var sourceLocations []srclocation.SourceLocation

	srcMapSlice := strings.Split(srcMap, ";")
	for i, instructionTuple := range srcMapSlice {
		var currentStruct srclocation.SourceLocation
		if i == 0 {
			currentStruct = srclocation.SourceLocation{}
		} else {
			currentStruct = sourceLocations[len(sourceLocations)-1]
		}
		for j, val := range strings.Split(instructionTuple, ":") {
			if val != "" {
				var err error
				// We do this because the split tuple might have any length <= 4.
				// Most of these cases won't be hit for most tuples.
				if j == 0 {
					currentStruct.ByteOffset, err = strconv.Atoi(val)
				} else if j == 1 {
					currentStruct.ByteLength, err = strconv.Atoi(val)
				} else if j == 2 {
					sourceFileIndex, err := strconv.Atoi(val)
					if err != nil {
						return sourceLocations, err
					}
					if sourceFileIndex != -1 {
						currentStruct.SourceFileName = srcList[sourceFileIndex]
					} else {
						currentStruct.SourceFileName = ""
					}
				} else if j == 3 {
					currentStruct.JumpType = rune(val[0])
				}
				if err != nil {
					return sourceLocations, err
				}
			}
		}
		sourceLocations = append(sourceLocations, currentStruct)
	}
	return sourceLocations, nil
}


















type CoverageLoc struct {
	HitCount      int
	CoverageRange srclocation.SourceLocation
	SrcLocs       []srclocation.SourceLocation
}

func GetCoverageLocs(filename string) ([]CoverageLoc, error) {
	plainAST, err := ast.Get(filename)
	if err != nil {
		return []CoverageLoc{}, err
	}

	covLocs, err := ToCoverageLocs(plainAST)
	if err != nil {
		return []CoverageLoc{}, err
	}
	return covLocs, nil
	// sort coverageLocs by .CoverageRange.ByteOffset
}

func ToCoverageLocs(node ast.ASTTree) ([]CoverageLoc, error) {
	var covLocs []CoverageLoc
	var covLoc CoverageLoc
	covLoc.CoverageRange = node.SrcLoc

	covLoc.SrcLocs = append(covLoc.SrcLocs, node.SrcLoc)
	children := node.Children
	sort.Slice(children, func(i, j int) bool {
		// If they have the same byte offset, my current belief is that that can
    	// only happen if one of them is empty. In that case, we're going to throw
    	// it away anyway.
		return children[i].SrcLoc.ByteOffset < children[j].SrcLoc.ByteOffset
	})
	for _, childNode := range children {
		if childNode.SrcLoc.ByteLength == 0 {
			// This node has no representation in the source file and neither could its children.
			continue
		} else if node.SrcLoc.ByteLength == childNode.SrcLoc.ByteLength {
			// This node is equal in size to its parent. Any siblings must be empty.
			// Not sure what to do here yet, let's roll with it and see what happens.
		}

		rightSrcLoc := covLoc.SrcLocs[len(covLoc.SrcLocs)-1]

		byteOffset1 := rightSrcLoc.ByteOffset
		byteLength1 := childNode.SrcLoc.ByteOffset - rightSrcLoc.ByteOffset
		if byteLength1 < 0 {
			return covLocs, errors.New("Negative byte length1")
		}
		byteOffset2 := childNode.SrcLoc.ByteOffset + childNode.SrcLoc.ByteLength
		byteLength2 := rightSrcLoc.ByteOffset + rightSrcLoc.ByteLength - byteOffset2
		if byteLength2 < 0 {
			return covLocs, errors.New("Negative byte length2")
		}

		covLoc.SrcLocs = append(
			// Cut off the last one, since we just split it in two.
			covLoc.SrcLocs[:len(covLoc.SrcLocs)-1],
			srclocation.SourceLocation{
				byteOffset1,
				byteLength1,
				node.SrcLoc.SourceFileName,
				*new(rune),
			},
			srclocation.SourceLocation{
				byteOffset2,
				byteLength2,
				node.SrcLoc.SourceFileName,
				*new(rune),
			},
		)

		childCovLocs, err := ToCoverageLocs(*childNode)
		if err == nil {
			covLocs = append(covLocs, childCovLocs...)
		}

	}
	covLocs = append(covLocs, covLoc)
	// TODO: sort coverLocs
	return covLocs, nil
}
