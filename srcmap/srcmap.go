package srcmap

import (
	"strconv"
	"strings"

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
